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

type ZigbeeDeviceImpl struct {
	log    *log.Context
	client *wh.Client
	added  bool

	baseUrl      string
	friendlyName string
	requests     func(ZigbeeRequest)

	dev    *devices.Device
	info   *services.Info
	online *services.Online

	update      *WrapperUpdate
	action      *WrapperAction
	battery     *WrapperBattery
	climate     *WrapperClimate
	light       *WrapperLight
	environment *WrapperEnvironment
	contact     *WrapperContact
	cover       *WrapperCover
	generic     *WrapperGeneric
}

func NewZigbeeDeviceImpl(info DeviceInfo, client *wh.Client, baseUrl string, requests func(ZigbeeRequest)) *ZigbeeDeviceImpl {
	dev := &ZigbeeDeviceImpl{
		log:      log.NewContext(log.DefaultLogger, info.IEEEAddress, log.DebugLevel),
		client:   client,
		baseUrl:  baseUrl,
		requests: requests,
		dev:      devices.NewDevice(info.IEEEAddress, clientsapi.Device_GENERIC),
		info:     services.NewInfo(),
		online:   services.NewOnline(),
	}

	dev.dev.AddService(
		dev.info,
		dev.online,
	)

	dev.log.Infof("created device")

	dev.UpdateInfo(info)

	return dev
}

func (dev *ZigbeeDeviceImpl) Device() *devices.Device { return dev.dev }
func (dev *ZigbeeDeviceImpl) Name() string            { return dev.friendlyName }

func (dev *ZigbeeDeviceImpl) sendZigbeeRequest(payload []byte) {
	dev.requests(ZigbeeRequest{Topic: dev.friendlyName + "/set", Payload: payload})
}

func (dev *ZigbeeDeviceImpl) UpdateInfo(info DeviceInfo) {
	dev.friendlyName = info.FriendlyName
	dev.info.Name.Set(info.FriendlyName)
	dev.info.Model.Set(info.ModelID)
	dev.info.Manufacturer.Set(info.Manufacturer)
	dev.info.SerialNumber.Set(info.IEEEAddress)
	dev.info.FirmwareVersion.Set(info.SoftwareBuildID)
	dev.info.WebUrl.Set(fmt.Sprintf("http://%s/#/device/%s/info", dev.baseUrl, info.IEEEAddress))

	dev.log.Debugf("info: %v", info)

	var handled []HandledExpose

	if dev.update == nil && SupportsUpdate(info) {
		dev.update = NewWrapperUpdate(dev.log, dev.dev, dev.sendZigbeeRequest)
	}
	if dev.update != nil {
		handled = append(handled, dev.update.UpdateInfo(info)...)
	}

	if dev.action == nil && SupportsAction(info) {
		dev.action = NewWrapperAction(dev.log, dev.dev)
	}
	if dev.action != nil {
		handled = append(handled, dev.action.UpdateInfo(info)...)
	}

	if dev.battery == nil && SupportsBattery(info) {
		dev.battery = NewWrapperBattery(dev.log, dev.dev)
	}
	if dev.battery != nil {
		handled = append(handled, dev.battery.UpdateInfo(info)...)
	}

	if dev.climate == nil && SupportsClimate(info) {
		dev.climate = NewWrapperClimate(dev.log, dev.dev, dev.sendZigbeeRequest)
	}
	if dev.climate != nil {
		handled = append(handled, dev.climate.UpdateInfo(info)...)
	}

	if dev.light == nil && SupportsLight(info) {
		dev.light = NewWrapperLight(dev.log, dev.dev, dev.sendZigbeeRequest)
	}
	if dev.light != nil {
		handled = append(handled, dev.light.UpdateInfo(info)...)
	}

	if dev.environment == nil && SupportsEnvironment(info) {
		dev.environment = NewWrapperEnvironment(dev.log, dev.dev)
	}
	if dev.environment != nil {
		handled = append(handled, dev.environment.UpdateInfo(info)...)
	}

	if dev.contact == nil && SupportsContact(info) {
		dev.contact = NewWrapperContact(dev.log, dev.dev)
	}
	if dev.contact != nil {
		handled = append(handled, dev.contact.UpdateInfo(info)...)
	}

	if dev.cover == nil && SupportsCover(info) {
		dev.cover = NewWrapperCover(dev.log, dev.dev, dev.sendZigbeeRequest)
	}
	if dev.cover != nil {
		handled = append(handled, dev.cover.UpdateInfo(info)...)
	}

	// Add unhandled properties to the generic service.
	if dev.generic == nil && len(handled) < len(info.Definition.Exposes) {
		dev.generic = NewWrapperGeneric(dev.log, dev.dev)
	}
	if dev.generic != nil {
		handled = append(handled, dev.generic.UpdateInfo(info, handled)...)
	}

	// Check for unsupported expose types.
	for _, expose := range info.Definition.Exposes {
		if !slices.Contains(handled, HandledExpose{expose.Type, expose.Property}) {
			dev.log.Warnf("unsupported expose type %q: %s", expose.Type, expose)
		}
	}
}

func (dev *ZigbeeDeviceImpl) UpdateOnline(online bool) {
	dev.online.Online.Set(online)
}

func (dev *ZigbeeDeviceImpl) UpdateState(state DeviceState) {
	dev.log.Debugf("state: %v", state)

	dev.online.LastSeen.Set(state.LastSeen)

	var handled []string
	if dev.update != nil {
		handled = append(handled, dev.update.UpdateState(state)...)
	}
	if dev.action != nil {
		handled = append(handled, dev.action.UpdateState(state)...)
	}
	if dev.battery != nil {
		handled = append(handled, dev.battery.UpdateState(state)...)
	}
	if dev.climate != nil {
		handled = append(handled, dev.climate.UpdateState(state)...)
	}
	if dev.light != nil {
		handled = append(handled, dev.light.UpdateState(state)...)
	}
	if dev.environment != nil {
		handled = append(handled, dev.environment.UpdateState(state)...)
	}
	if dev.contact != nil {
		handled = append(handled, dev.contact.UpdateState(state)...)
	}
	if dev.cover != nil {
		handled = append(handled, dev.cover.UpdateState(state)...)
	}
	if dev.generic != nil {
		handled = append(handled, dev.generic.UpdateState(state, handled)...)
	}

	// Check for unhandled properties.
	for key, value := range state.Values {
		if !slices.Contains(handled, key) {
			dev.log.Errorf("unsupported state property %q: %s", key, value)

			// // Add unhandled properties to the generic service.
			// if dev.generic == nil {
			// 	dev.generic = NewWrapperGeneric(dev.log, dev.dev)
			// }
			// handled = append(handled, dev.generic.UpdateState(state, handled)...)
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
