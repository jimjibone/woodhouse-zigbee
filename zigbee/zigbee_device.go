package zigbee

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/wh/v1/bridges"
	"github.com/jimjibone/wh/v1/bridges/services"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
)

type ZigbeeDeviceImpl struct {
	log    *log.Context
	bridge *bridges.Bridge
	added  bool

	baseUrl      string
	friendlyName string
	requests     func(ZigbeeRequest)
	correlator   *ResponseCorrelator

	dev    *bridges.Device
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
	motion      *WrapperMotion
	presence    *WrapperPresence
	generic     *WrapperGeneric
}

func NewZigbeeDeviceImpl(info DeviceInfo, client *bridges.Bridge, baseUrl string, requests func(ZigbeeRequest), correlator *ResponseCorrelator) *ZigbeeDeviceImpl {
	dev := &ZigbeeDeviceImpl{
		log:        log.NewContext(log.DefaultLogger, info.IEEEAddress, log.DebugLevel),
		bridge:     client,
		baseUrl:    baseUrl,
		requests:   requests,
		correlator: correlator,
		dev:        bridges.NewDevice(info.IEEEAddress, clientsapi.Device_DEVICE),
		info:       services.NewInfo(),
		online:     services.NewOnline(),
	}

	dev.dev.AddService(
		dev.info,
		dev.online,
	)

	dev.info.EnableRename(dev.handleRename)

	dev.log.Infof("created device")

	dev.UpdateInfo(info)

	return dev
}

func (dev *ZigbeeDeviceImpl) Device() *bridges.Device { return dev.dev }
func (dev *ZigbeeDeviceImpl) Name() string            { return dev.friendlyName }

func (dev *ZigbeeDeviceImpl) sendZigbeeRequest(payload []byte) {
	dev.requests(ZigbeeRequest{Topic: dev.friendlyName + "/set", Payload: payload})
}

func (dev *ZigbeeDeviceImpl) sendUpdateRequest() {
	dev.requests(ZigbeeRequest{
		Topic:   "bridge/request/device/ota_update/update",
		Payload: fmt.Appendf(nil, `{"id": "%s"}`, dev.friendlyName),
	})
}

// handleRename forwards a rename request to zigbee2mqtt and waits for its
// response. The device's friendly name and info attributes are not updated
// here; zigbee2mqtt republishes bridge/devices after a successful rename and
// UpdateInfo picks up the new name from there.
func (dev *ZigbeeDeviceImpl) handleRename(newName string) error {
	if dev.correlator == nil {
		return fmt.Errorf("rename not supported")
	}

	tx, ch := dev.correlator.NewTransaction()
	defer dev.correlator.Forget(tx)

	payload, err := json.Marshal(struct {
		From                string `json:"from"`
		To                  string `json:"to"`
		HomeassistantRename bool   `json:"homeassistant_rename"`
		Transaction         string `json:"transaction"`
	}{
		From:        dev.friendlyName,
		To:          newName,
		Transaction: tx,
	})
	if err != nil {
		return err
	}

	dev.log.Infof("renaming %q to %q (transaction %s)", dev.friendlyName, newName, tx)
	dev.requests(ZigbeeRequest{
		Topic:   "bridge/request/device/rename",
		Payload: payload,
	})

	select {
	case res := <-ch:
		if !res.OK {
			return fmt.Errorf("zigbee2mqtt rename failed: %s", res.Error)
		}
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timed out waiting for zigbee2mqtt rename response")
	}
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
		dev.update = NewWrapperUpdate(dev.log, dev.dev, dev.sendUpdateRequest)
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

	if dev.motion == nil && SupportsMotion(info) {
		dev.motion = NewWrapperMotion(dev.log, dev.dev)
	}
	if dev.motion != nil {
		handled = append(handled, dev.motion.UpdateInfo(info)...)
	}

	if dev.presence == nil && SupportsPresence(info) {
		dev.presence = NewWrapperPresence(dev.log, dev.dev)
	}
	if dev.presence != nil {
		handled = append(handled, dev.presence.UpdateInfo(info)...)
	}

	// Add unhandled properties to the generic service.
	if dev.generic == nil && len(handled) < len(info.Definition.Exposes) {
		dev.generic = NewWrapperGeneric(dev.log, dev.dev, dev.sendZigbeeRequest)
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
	if dev.motion != nil {
		handled = append(handled, dev.motion.UpdateState(state)...)
	}
	if dev.presence != nil {
		handled = append(handled, dev.presence.UpdateState(state)...)
	}

	// Handle generic last which sweeps up anything not already handled.
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
		err := dev.bridge.AddDevice(dev.dev)
		if err != nil {
			dev.log.Fatalf("failed to add device: %s", err)
		}
	}
}
