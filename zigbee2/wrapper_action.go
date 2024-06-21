package zigbee

import (
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperAction)(nil)

type WrapperAction struct {
	log    *log.Context
	button *services.Button

	actionProperty  string
	actionConverter *EnumConverter
}

func SupportsAction(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "enum":
			switch {
			case expose.Property == "action" && expose.Category == "diagnostic":
				// This is the feature we need.
				return true
			}
		}
	}
	return false
}

func NewWrapperAction(log *log.Context, dev *devices.Device) *WrapperAction {
	wrapper := &WrapperAction{
		log:    log,
		button: services.NewButton(""),
	}
	dev.AddService(wrapper.button)
	return wrapper
}

func (wrapper *WrapperAction) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	var err error
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "enum":
			switch {
			case expose.Property == "action" && expose.Category == "diagnostic":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.actionProperty = expose.Property
				wrapper.actionConverter, err = UnmarshalEnum(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal action value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("action value expose %q: %s", wrapper.actionProperty, wrapper.actionConverter)
				}
				wrapper.button.State.SetOptions(wrapper.actionConverter.Values)
			}
		}
	}
	return handled
}

func (wrapper *WrapperAction) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		switch key {
		case wrapper.actionProperty:
			handled = append(handled, key)
			val, err := wrapper.actionConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal action value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("action value %q: %s", wrapper.actionProperty, val)
				wrapper.button.State.Set(val)
			}
		}
	}
	return handled
}
