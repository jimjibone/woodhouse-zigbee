package zigbee

import (
	"fmt"
	"slices"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

type ZigbeeLight struct {
	log    *log.Context
	client *wh.Client
	added  bool

	baseUrl      string
	friendlyName string
	requests     func(ZigbeeRequest)

	dev    *devices.Device
	info   *services.Info
	online *services.Online

	light *WrapperLight
}

func NewZigbeeLight(info DeviceInfo, client *wh.Client, baseUrl string, requests func(ZigbeeRequest)) *ZigbeeLight {
	dev := &ZigbeeLight{
		log:      log.NewContext(log.DefaultLogger, info.IEEEAddress, log.DebugLevel),
		client:   client,
		baseUrl:  baseUrl,
		requests: requests,
		dev:      devices.NewDevice(info.IEEEAddress, clientsapi.Device_LIGHTBULB),
		info:     services.NewInfo(),
		online:   services.NewOnline(),
	}

	dev.dev.AddService(
		dev.info,
		dev.online,
	)

	dev.log.Infof("created lightbulb")

	dev.UpdateInfo(info)

	return dev
}

func (dev *ZigbeeLight) Device() *devices.Device { return dev.dev }
func (dev *ZigbeeLight) Name() string            { return dev.friendlyName }

func (dev *ZigbeeLight) UpdateInfo(info DeviceInfo) {
	dev.friendlyName = info.FriendlyName
	dev.info.Name.Set(info.FriendlyName)
	dev.info.Model.Set(info.ModelID)
	dev.info.Manufacturer.Set(info.Manufacturer)
	dev.info.SerialNumber.Set(info.IEEEAddress)
	dev.info.FirmwareVersion.Set(info.SoftwareBuildID)
	dev.info.WebUrl.Set(fmt.Sprintf("http://%s/#/device/%s/info", dev.baseUrl, info.IEEEAddress))

	dev.log.Debugf("info: %v", info)

	var handled []HandledExpose

	if dev.light == nil && SupportsLight(info) {
		dev.light = NewWrapperLight(dev.log, dev.dev, func(payload []byte) {
			dev.requests(ZigbeeRequest{Topic: dev.friendlyName + "/set", Payload: payload})
		})
	}
	if dev.light != nil {
		handled = append(handled, dev.light.UpdateInfo(info)...)
	}

	// Check for unsupported expose types.
	for _, expose := range info.Definition.Exposes {
		if !slices.Contains(handled, HandledExpose{expose.Type, expose.Property}) {
			dev.log.Warnf("unsupported expose type %q: %s", expose.Type, expose)
		}
	}
}

func (dev *ZigbeeLight) UpdateOnline(online bool) {
	dev.online.Online.Set(online)
}

func (dev *ZigbeeLight) UpdateState(state DeviceState) {
	dev.log.Debugf("state: %v", state)

	dev.online.LastSeen.Set(state.LastSeen)

	var handled []string
	if dev.light != nil {
		handled = append(handled, dev.light.UpdateState(state)...)
	}

	// Check for unhandled properties.
	for key, value := range state.Values {
		if !slices.Contains(handled, key) {
			dev.log.Errorf("unsupported state property %q: %s", key, value)
		}
	}

	// Add this device to the client if not done already.
	if !dev.added {
		dev.added = true
		err := dev.client.AddDevice(dev.dev)
		if err != nil {
			dev.log.Fatalf("failed to add device: %s", err)
		}
	}
}
