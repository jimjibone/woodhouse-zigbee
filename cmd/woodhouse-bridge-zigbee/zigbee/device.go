package zigbee

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-zigbee/zigbee/converters"
	"github.com/jimjibone/woodhouse-4/wh"
	"google.golang.org/protobuf/proto"
)

type ZigbeeDevice struct {
	Added          bool
	comms          *wh.BridgeComms
	id             string
	name           string
	description    string
	online         bool
	lastSeen       time.Time
	converters     map[string]converters.Converter
	values         map[string]*api.DeviceValue
	requestHandler func(topic string, payload []byte)
}

func NewZigbeeDevice(requestHandler func(topic string, payload []byte)) *ZigbeeDevice {
	zd := &ZigbeeDevice{
		converters:     make(map[string]converters.Converter),
		values:         make(map[string]*api.DeviceValue),
		requestHandler: requestHandler,
	}
	return zd
}

func (zd *ZigbeeDevice) String() string {
	return fmt.Sprintf("id: %s, name: %s, desc: %s, converters: %s, values: %s", zd.id, zd.name, zd.description, zd.converters, zd.values)
}

func (zd *ZigbeeDevice) ID() string   { return "zigbee" + zd.id }
func (zd *ZigbeeDevice) Name() string { return zd.name }

func (zd *ZigbeeDevice) Init(comms *wh.BridgeComms) { zd.comms = comms }

func (zd *ZigbeeDevice) SendFullUpdate() {
	err := zd.comms.SendInfo(&api.DeviceInfo{
		DeviceId:    zd.ID(),
		Name:        zd.name,
		Description: zd.description,
	})
	if err != nil {
		log.Printf("ERROR: device %s failed to send info: %s", zd.id, err)
	}

	msg := &api.DeviceState{
		DeviceId:   zd.ID(),
		Online:     zd.online,
		LastSeen:   apitools.TimeToTimestamp(zd.lastSeen),
		FullUpdate: true,
		Values:     []*api.DeviceValue{},
	}
	for name, val := range zd.values {
		val.Name = name
		msg.Values = append(msg.Values, proto.Clone(val).(*api.DeviceValue))
	}
	err = zd.comms.SendState(msg)
	if err != nil {
		log.Printf("ERROR: device %s failed to send state: %s", zd.id, err)
	}
}

func (zd *ZigbeeDevice) HandleRequest(req *api.DeviceRequest) error {
	log.Printf("device %s handling request: %s", zd.id, req)
	if zd.requestHandler != nil {
		reqjson := make(map[string]json.RawMessage)
		for _, val := range req.Values {
			if conv, found := zd.converters[val.Name]; found {
				valjson, err := conv.Marshal(val)
				if err != nil {
					return fmt.Errorf("marshal %s: %s", val, err)
				}
				reqjson[val.Name] = valjson
			}
		}
		log.Printf("device %s handling request: %s", zd.id, reqjson)
		if len(reqjson) > 0 {
			data, err := json.Marshal(reqjson)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal reqjson: %s --- %s", reqjson, err))
			}
			zd.requestHandler(zd.name+"/set", data)
		}
	}
	return nil
}

func (zd *ZigbeeDevice) UpdateInfo(info DeviceInfo) error {
	zd.id = info.IEEEAddress
	zd.name = info.FriendlyName
	zd.description = info.Definition.Description

	zd.converters = make(map[string]converters.Converter)
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "binary":
			conv, err := converters.NewBinary(expose.Data)
			if err != nil {
				return fmt.Errorf("binary %q: %w", expose.Property, err)
			} else {
				zd.converters[expose.Property] = conv
			}

		case "numeric":
			conv, err := converters.NewNumeric(expose.Data)
			if err != nil {
				return fmt.Errorf("numeric %q: %w", expose.Property, err)
			} else {
				zd.converters[expose.Property] = conv
			}

		case "enum":
			conv, err := converters.NewEnum(expose.Data)
			if err != nil {
				return fmt.Errorf("enum %q: %w", expose.Property, err)
			} else {
				zd.converters[expose.Property] = conv
			}

		case "text":
			conv, err := converters.NewText(expose.Data)
			if err != nil {
				return fmt.Errorf("text %q: %w", expose.Property, err)
			} else {
				zd.converters[expose.Property] = conv
			}

		case "light", "climate", "switch", "fan", "cover", "lock":
			convs, err := converters.NewFeature(expose.Data)
			if err != nil {
				return fmt.Errorf("feature %q: %w", expose.Type, err)
			} else {
				for property, conv := range convs {
					zd.converters[property] = conv
				}
			}

		default:
			return fmt.Errorf("unknown value type %q", expose.Type)
		}
	}

	if zd.comms != nil {
		err := zd.comms.SendInfo(&api.DeviceInfo{
			DeviceId:    zd.ID(),
			Name:        zd.name,
			Description: zd.description,
		})
		if err != nil {
			log.Printf("ERROR: device %s failed to send info: %s", zd.id, err)
		}
	}
	return nil
}

func (zd *ZigbeeDevice) UpdateOnline(online bool) {
	if zd.online != online {
		zd.online = online

		log.Printf("device %s online: %t", zd.id, online)

		msg := &api.DeviceState{
			DeviceId:   zd.ID(),
			Online:     zd.online,
			LastSeen:   apitools.TimeToTimestamp(zd.lastSeen),
			FullUpdate: false,
		}
		err := zd.comms.SendState(msg)
		if err != nil {
			log.Printf("ERROR: device %s failed to send state: %s", zd.id, err)
		}
	}
}

func (zd *ZigbeeDevice) UpdateState(state DeviceState) error {
	changed := false
	if !zd.lastSeen.After(state.LastSeen) {
		changed = true
		zd.lastSeen = state.LastSeen
	}

	for name, value := range state.Values {
		if converter, found := zd.converters[name]; found {
			// Use the converter to convert this state value to a woodhouse value.
			next, err := converter.Unmarshal(value)
			if err != nil {
				log.Printf("ERROR: device %s failed to convert value %q with %s: %s", zd.id, name, value, err)
			} else {
				if prev, found := zd.values[name]; found {
					log.Printf("device %s updated value %q: %v --> %v (converted)", zd.id, name, prev, next)
				} else {
					log.Printf("device %s new value %q: %v (converted)", zd.id, name, next)
				}
				changed = true
				zd.values[name] = next
			}
		} else {
			// Do a direct conversion.
			var val interface{}
			err := json.Unmarshal(value, &val)
			if err != nil {
				log.Printf("ERROR: device %s failed to unmarshal value %q with %s", zd.id, name, value)
			} else {
				var next *api.DeviceValue
				switch v := val.(type) {
				case bool:
					next = converters.ConvertBool(v)

				case float64:
					next = converters.ConvertNumber(v)

				case string:
					next = converters.ConvertText(v)

				case nil:
					// Ignore.

				default:
					switch name {
					case "update":
						// Ignore.

					default:
						log.Printf("ERROR: device %s failed to convert value %q with %+v: no converter", zd.id, name, val)
					}
				}
				if next != nil {
					if prev, found := zd.values[name]; found {
						log.Printf("device %s updated value %q: %v --> %v", zd.id, name, prev, next)
					} else {
						log.Printf("device %s new value %q: %v", zd.id, name, next)
					}
					changed = true
					zd.values[name] = next
				}
			}
		}
	}

	if changed && zd.comms != nil {
		msg := &api.DeviceState{
			DeviceId:   zd.ID(),
			Online:     zd.online,
			LastSeen:   apitools.TimeToTimestamp(zd.lastSeen),
			FullUpdate: true,
			Values:     []*api.DeviceValue{},
		}
		for name, val := range zd.values {
			val.Name = name
			msg.Values = append(msg.Values, proto.Clone(val).(*api.DeviceValue))
		}
		err := zd.comms.SendState(msg)
		if err != nil {
			log.Printf("ERROR: device %s failed to send state: %s", zd.id, err)
		}
	}

	return nil
}
