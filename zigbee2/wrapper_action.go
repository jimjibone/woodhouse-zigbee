package zigbee

import (
	"strings"
	"time"

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

	actionDurationProperty  string
	actionDurationConverter *NumericConverter
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

		case "numeric":
			switch {
			case expose.Property == "action_duration" && expose.Category == "diagnostic":
				// We also like this but don't need it.
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
				if !wrapper.button.State.IsSet() {
					wrapper.button.State.Set("")
				}
				wrapper.button.State.SetOptions(wrapper.actionConverter.Values)
			}

		case "numeric":
			switch {
			case expose.Property == "action_duration" && expose.Category == "diagnostic":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.actionDurationProperty = expose.Property
				wrapper.actionDurationConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal action_duration value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("action_duration value expose %q: %s", wrapper.actionDurationProperty, wrapper.actionDurationConverter)
				}
				if !wrapper.button.Duration.IsSet() {
					wrapper.button.Duration.Set(0)
				}
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

				// Reset the action duration if the current state doesn't contain '_hold'.
				if !strings.Contains(val, "_hold") {
					wrapper.button.Duration.Set(0)
				}
			}

		case wrapper.actionDurationProperty:
			handled = append(handled, key)
			val, err := wrapper.actionDurationConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal action duration value %q: %s", value, err)
			} else {
				d := time.Duration(val * float64(time.Second))
				wrapper.log.Debugf("action duration value %q: %f --> %s", wrapper.actionDurationProperty, val, d)
				wrapper.button.Duration.Set(d)
			}
		}
	}
	return handled
}
