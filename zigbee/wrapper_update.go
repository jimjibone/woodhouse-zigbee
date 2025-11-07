package zigbee

import (
	"encoding/json"
	"fmt"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperUpdate)(nil)

type WrapperUpdate struct {
	log           *log.Context
	requestUpdate func()

	update *services.Update

	updateAvailableConverter *BinaryConverter
}

type updateConverter struct {
	InstalledVersion json.RawMessage `json:"installed_version"`
	LatestVersion    json.RawMessage `json:"latest_version"`
	State            string          `json:"state"`     // "available", "updating", "idle"
	Progress         *float32        `json:"progress"`  // 7.01 (%)
	Remaining        *float32        `json:"remaining"` // 952 (seconds)
}

func SupportsUpdate(info DeviceInfo) bool {
	return info.Definition.SupportsOTA
}

func NewWrapperUpdate(log *log.Context, dev *devices.Device, requests func()) *WrapperUpdate {
	wrapper := &WrapperUpdate{
		log:           log,
		update:        services.NewUpdate(""),
		requestUpdate: requests,

		updateAvailableConverter: &BinaryConverter{
			ValueOn:  "true",
			ValueOff: "false",
		},
	}
	wrapper.update.OnAction(wrapper.handleAction)
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
				switch conv.State {
				case "available":
				case "updating":
					wrapper.update.Updating.Set(true)
				case "idle":
					wrapper.update.Updating.Set(false)
				default:
					wrapper.log.Errorf("unsupported state %q", conv.State)
					wrapper.update.Updating.Set(false)
				}
				if conv.Progress != nil {
					wrapper.update.Progress.Set(int64(*conv.Progress))
				} else {
					wrapper.update.Progress.Set(0)
				}
				if conv.Remaining != nil {
					wrapper.update.Remaining.Set(time.Duration(*conv.Remaining) * time.Second)
				} else {
					wrapper.update.Remaining.Set(0)
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperUpdate) handleAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	wrapper.log.Debugf("handling request: %s", request)
	if wrapper.requestUpdate != nil {
		for _, val := range request.Values {
			switch val.Id {
			case wrapper.update.StartUpdate.ID():
				// Updates work differently to standard device requests.
				wrapper.log.Debugf("handling request: update")
				wrapper.requestUpdate()

			default:
				wrapper.log.Errorf("unsupported request value: %s", val)
				return fmt.Errorf("unsupported request value: %s", val)
			}
		}
	}
	return nil
}
