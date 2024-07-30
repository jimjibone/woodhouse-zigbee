package zigbee

import (
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperBattery)(nil)

type WrapperBattery struct {
	log     *log.Context
	battery *services.Battery

	batteryProperty  string
	batteryConverter *NumericConverter

	voltageProperty  string
	voltageConverter *NumericConverter
}

func SupportsBattery(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "numeric":
			switch {
			case expose.Property == "battery" && expose.Category == "diagnostic":
				// This is the feature we need.
				return true

			case expose.Property == "voltage" && expose.Category == "diagnostic":
				// Not needed but used.
			}
		}
	}
	return false
}

func NewWrapperBattery(log *log.Context, dev *devices.Device) *WrapperBattery {
	wrapper := &WrapperBattery{
		log:     log,
		battery: services.NewBattery(""),
	}
	dev.AddService(wrapper.battery)
	return wrapper
}

func (wrapper *WrapperBattery) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	var err error
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "numeric":
			switch {
			case expose.Property == "battery" && expose.Category == "diagnostic":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.batteryProperty = expose.Property
				wrapper.batteryConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal battery level value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("battery level value expose %q: %s", wrapper.batteryProperty, wrapper.batteryConverter)
				}
				if !wrapper.battery.Level.IsSet() {
					wrapper.battery.Level.Set(0)
				}

			case expose.Property == "voltage" && expose.Category == "diagnostic":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.voltageProperty = expose.Property
				wrapper.voltageConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal battery voltage value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("battery voltage value expose %q: %s", wrapper.voltageProperty, wrapper.voltageConverter)
				}
				if wrapper.voltageConverter.ValueMin != nil && wrapper.voltageConverter.ValueMax != nil && wrapper.voltageConverter.ValueStep != nil {
					wrapper.battery.Voltage.SetLimits(*wrapper.voltageConverter.ValueMin, *wrapper.voltageConverter.ValueMax, *wrapper.voltageConverter.ValueStep)

					if !wrapper.battery.Voltage.IsSet() {
						wrapper.battery.Voltage.Set(*wrapper.voltageConverter.ValueMin)
					}
				} else {
					if !wrapper.battery.Voltage.IsSet() {
						wrapper.battery.Voltage.Set(0)
					}
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperBattery) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		switch key {
		case wrapper.batteryProperty:
			handled = append(handled, key)
			val, err := wrapper.batteryConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal battery level value %q: %s", value, err)
			} else {
				level := int64(val)
				wrapper.log.Debugf("battery level value %q: %f -> %d", wrapper.batteryProperty, val, level)
				wrapper.battery.Level.Set(level)
			}

		case wrapper.voltageProperty:
			handled = append(handled, key)
			val, err := wrapper.voltageConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal battery voltage value %q: %s", value, err)
			} else {
				val = val / 1000.0 // millivolts to volts
				wrapper.log.Debugf("battery voltage value %q: %v", wrapper.voltageProperty, val)
				wrapper.battery.Voltage.Set(val)
			}
		}
	}
	return handled
}
