package zigbee

import (
	"encoding/json"

	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperUpdate)(nil)

type WrapperUpdate struct {
	log      *log.Context
	requests func(payload []byte)

	update *services.Update

	updateAvailableConverter *BinaryConverter
}

type updateConverter struct {
	InstalledVersion json.RawMessage `json:"installed_version"`
	LatestVersion    json.RawMessage `json:"latest_version"`
	State            string          `json:"state"`
}

func SupportsUpdate(info DeviceInfo) bool {
	return info.Definition.SupportsOTA
}

func NewWrapperUpdate(log *log.Context, dev *devices.Device, requests func(payload []byte)) *WrapperUpdate {
	wrapper := &WrapperUpdate{
		log:      log,
		update:   services.NewUpdate(""),
		requests: requests,

		updateAvailableConverter: &BinaryConverter{
			ValueOn:  "true",
			ValueOff: "false",
		},
	}
	// wrapper.update.OnAction(wrapper.handleAction)
	dev.AddService(wrapper.update)
	return wrapper
}

func (wrapper *WrapperUpdate) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	// No info to be consumed for this one.
	if !wrapper.update.Available.IsSet() {
		wrapper.update.Available.Set(false)
	}
	return handled
}

func (wrapper *WrapperUpdate) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		switch key {
		case "update_available":
			handled = append(handled, key)
			val, err := wrapper.updateAvailableConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal update_available value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("update_available value: %v", val)
				wrapper.update.Available.Set(val)
			}

		case "update":
			handled = append(handled, key)
			conv := updateConverter{}
			err := json.Unmarshal(value, &conv)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal update value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("update current: %s, latest: %s, state: %s", conv.InstalledVersion, conv.LatestVersion, conv.State)
				wrapper.update.CurrentVersion.Set(string(conv.InstalledVersion))
				wrapper.update.UpdateVersion.Set(string(conv.LatestVersion))
				// wrapper.update.State.Set(conv.State)
			}

		}
	}
	return handled
}
