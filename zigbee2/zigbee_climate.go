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

type ZigbeeClimate struct {
	log    *log.Context
	client *wh.Client
	added  bool

	baseUrl      string
	friendlyName string
	requests     func(ZigbeeRequest)

	dev    *devices.Device
	info   *services.Info
	online *services.Online

	climate *WrapperClimate
	battery *WrapperBattery
}

func NewZigbeeClimate(info DeviceInfo, client *wh.Client, baseUrl string, requests func(ZigbeeRequest)) *ZigbeeClimate {
	dev := &ZigbeeClimate{
		log:      log.NewContext(log.DefaultLogger, info.IEEEAddress, log.DebugLevel),
		client:   client,
		baseUrl:  baseUrl,
		requests: requests,
		dev:      devices.NewDevice(info.IEEEAddress, clientsapi.Device_CLIMATE),
		info:     services.NewInfo(),
		online:   services.NewOnline(),
	}

	dev.dev.AddService(
		dev.info,
		dev.online,
	)

	dev.log.Infof("created climate")

	dev.UpdateInfo(info)

	return dev
}

func (dev *ZigbeeClimate) Device() *devices.Device { return dev.dev }
func (dev *ZigbeeClimate) Name() string            { return dev.friendlyName }

func (dev *ZigbeeClimate) UpdateInfo(info DeviceInfo) {
	dev.friendlyName = info.FriendlyName
	dev.info.Name.Set(info.FriendlyName)
	dev.info.Model.Set(info.ModelID)
	dev.info.Manufacturer.Set(info.Manufacturer)
	dev.info.SerialNumber.Set(info.IEEEAddress)
	dev.info.FirmwareVersion.Set(info.SoftwareBuildID)
	dev.info.WebUrl.Set(fmt.Sprintf("http://%s/#/device/%s/info", dev.baseUrl, info.IEEEAddress))

	dev.log.Debugf("info: %v", info)

	var handled []HandledExpose

	if dev.climate == nil && SupportsClimate(info) {
		dev.climate = NewWrapperClimate(dev.log, dev.dev, func(payload []byte) {
			dev.requests(ZigbeeRequest{Topic: dev.friendlyName + "/set", Payload: payload})
		})
	}
	if dev.climate != nil {
		handled = append(handled, dev.climate.UpdateInfo(info)...)
	}

	if dev.battery == nil && SupportsBattery(info) {
		dev.battery = NewWrapperBattery(dev.log, dev.dev)
	}
	if dev.battery != nil {
		handled = append(handled, dev.battery.UpdateInfo(info)...)
	}

	// Check for unsupported expose types.
	for _, expose := range info.Definition.Exposes {
		if !slices.Contains(handled, HandledExpose{expose.Type, expose.Property}) {
			dev.log.Warnf("unsupported expose type %q: %s", expose.Type, expose)
		}
	}
}

func (dev *ZigbeeClimate) UpdateOnline(online bool) {
	dev.online.Online.Set(online)
}

func (dev *ZigbeeClimate) UpdateState(state DeviceState) {
	dev.log.Debugf("state: %v", state)

	dev.online.LastSeen.Set(state.LastSeen)

	var handled []string
	if dev.climate != nil {
		handled = append(handled, dev.climate.UpdateState(state)...)
	}
	if dev.battery != nil {
		handled = append(handled, dev.battery.UpdateState(state)...)
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
