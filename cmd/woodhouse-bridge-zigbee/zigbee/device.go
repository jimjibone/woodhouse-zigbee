package zigbee

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/wh"
	"google.golang.org/protobuf/proto"
)

type ZigbeeDevice interface {
	wh.Device
	DeviceID() string
	IEEEAddress() string
	FriendlyName() string
	UpdateInfo(*DeviceInfo)
	UpdateState(*DeviceState)
}

func CreateDevice(info *DeviceInfo, publish func(topic string, payload []byte)) ZigbeeDevice {
	if info.Definition != nil {
		for _, exposed := range info.Definition.Exposes {
			if exposed.Name == "light" {
				return NewZigbeeDeviceLight(info, publish)
			}
		}
		return NewZigbeeDeviceLight(info, publish)
	}
	return nil
}

type ZigbeeDeviceLight struct {
	comms        *wh.BridgeComms
	ieeeAddress  string
	friendlyName string
	description  string
	lastSeen     time.Time
	exposers     map[string]*Exposed
	values       map[string]*api.DeviceValue
	publish      func(topic string, payload []byte)
}

func NewZigbeeDeviceLight(info *DeviceInfo, publish func(topic string, payload []byte)) *ZigbeeDeviceLight {
	zd := &ZigbeeDeviceLight{
		ieeeAddress: info.IEEEAddress,
		exposers:    make(map[string]*Exposed),
		values:      make(map[string]*api.DeviceValue),
		publish:     publish,
	}
	zd.updateInfo(info, false)
	return zd
}

func (zd *ZigbeeDeviceLight) Init(comms *wh.BridgeComms) { zd.comms = comms }
func (zd *ZigbeeDeviceLight) DeviceID() string           { return "zigbee" + zd.ieeeAddress }
func (zd *ZigbeeDeviceLight) IEEEAddress() string        { return zd.ieeeAddress }
func (zd *ZigbeeDeviceLight) FriendlyName() string       { return zd.friendlyName }

func (zd *ZigbeeDeviceLight) SendFullUpdate() {
	err := zd.comms.SendInfo(&api.DeviceInfo{
		DeviceId:    zd.DeviceID(),
		Name:        zd.friendlyName,
		Description: zd.description,
	})
	if err != nil {
		log.Printf("ERROR: device %s: failed to send info: %s", zd.friendlyName, err)
	}
	msg := &api.DeviceState{
		DeviceId:   zd.DeviceID(),
		FullUpdate: true,
		Values:     []*api.DeviceValue{},
	}
	for _, val := range zd.values {
		msg.Values = append(msg.Values, proto.Clone(val).(*api.DeviceValue))
	}
	err = zd.comms.SendState(msg)
	if err != nil {
		log.Printf("ERROR: device %s: failed to send state: %s", zd.friendlyName, err)
	}
}

func (zd *ZigbeeDeviceLight) HandleRequest(req *api.DeviceRequest) error {
	reqjson := make(map[string]interface{})
	for _, val := range req.Values {
		if exposer, found := zd.exposers[val.Name]; found {
			valjson := exposer.Value.GetJSON(val)
			if exposer.PrefixProperty != "" {
				if prev, found := reqjson[exposer.PrefixProperty]; found {
					switch p := prev.(type) {
					case map[string]interface{}:
						p[exposer.Property] = valjson
					default:
						panic(fmt.Sprintf("reqjson contains a non interface entry for a composite value: %s", exposer))
					}
				} else {
					reqjson[exposer.PrefixProperty] = map[string]interface{}{
						exposer.Property: valjson,
					}
				}
			} else {
				reqjson[exposer.Property] = valjson
			}
		}
	}
	if len(reqjson) > 0 {
		data, err := json.Marshal(reqjson)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal reqjson: %s --- %s", reqjson, err))
		}
		zd.publish(zd.friendlyName+"/set", data)
	}
	return nil
}

func (zd *ZigbeeDeviceLight) UpdateInfo(info *DeviceInfo) {
	zd.updateInfo(info, true)
}

func (zd *ZigbeeDeviceLight) updateInfo(info *DeviceInfo, sendChanges bool) {
	zd.friendlyName = info.FriendlyName
	if info.Definition != nil {
		zd.description = info.Definition.Description
		exposers, typeName := info.Definition.FlattenExposes()
		zd.exposers = exposers
		log.Printf("updating %s with %d exposers - type: %s", zd.friendlyName, len(zd.exposers), typeName)
		for name, exposer := range zd.exposers {
			log.Printf("updated exposer: %s --> %s", name, exposer)
		}
	}
	if sendChanges {
		err := zd.comms.SendInfo(&api.DeviceInfo{
			DeviceId:    zd.DeviceID(),
			Name:        zd.friendlyName,
			Description: zd.description,
		})
		if err != nil {
			log.Printf("ERROR: device %s: failed to send info: %s", zd.friendlyName, err)
		}
	}
}

func (zd *ZigbeeDeviceLight) UpdateState(state *DeviceState) {
	changed := false
	if !zd.lastSeen.Equal(state.LastSeen) {
		changed = true
		zd.lastSeen = state.LastSeen
	}

	// Flatten state values.
	stateValues := make(map[string]interface{})
	for name, value := range state.Values {
		switch val := value.(type) {
		case map[string]interface{}:
			for name2, value2 := range val {
				stateValues[name+"."+name2] = value2
			}
		default:
			stateValues[name] = val
		}
	}

	log.Printf("updating %s with %d values", zd.friendlyName, len(stateValues))

	// Convert state values to api values via the exposers.
	for name, val := range stateValues {
		if exposer, found := zd.exposers[name]; found {
			changed = true
			next := exposer.Value.GetValue(val)
			if next != nil {
				next.Name = name
				zd.values[name] = next
				log.Printf("updated value for state %q: %v --> %v", name, val, next)
			} else {
				log.Printf("ERROR: no value created for state %q: %v", name, val)
			}
		} else {
			log.Printf("ERROR: no exposer found for state %q: %v", name, val)
		}
	}

	if changed {
		msg := &api.DeviceState{
			DeviceId:   zd.DeviceID(),
			FullUpdate: true,
			Values:     []*api.DeviceValue{},
		}
		for _, val := range zd.values {
			msg.Values = append(msg.Values, proto.Clone(val).(*api.DeviceValue))
		}
		err := zd.comms.SendState(msg)
		if err != nil {
			log.Printf("ERROR: device %s: failed to send state: %s", zd.friendlyName, err)
		}
	}
}
