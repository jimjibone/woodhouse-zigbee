package zigbee

import (
	"github.com/jimjibone/log"
	"github.com/jimjibone/woodhouse-core/wh/v1/devices"
	"github.com/jimjibone/woodhouse-core/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperEnvironment)(nil)

type WrapperEnvironment struct {
	log         *log.Context
	environment *services.Environment

	temperatureProperty  string
	temperatureConverter *NumericConverter

	humidityProperty  string
	humidityConverter *NumericConverter

	pressureProperty  string
	pressureConverter *NumericConverter
}

func SupportsEnvironment(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "numeric":
			switch {
			case expose.Property == "temperature" ||
				expose.Property == "humidity" ||
				expose.Property == "pressure":
				// This is on of the features we need.
				return true
			}
		}
	}
	return false
}

func NewWrapperEnvironment(log *log.Context, dev *devices.Device) *WrapperEnvironment {
	wrapper := &WrapperEnvironment{
		log:         log,
		environment: services.NewEnvironment(""),
	}
	dev.AddService(wrapper.environment)
	return wrapper
}

func (wrapper *WrapperEnvironment) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	var err error
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "numeric":
			switch {
			case expose.Property == "temperature":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.temperatureProperty = expose.Property
				wrapper.temperatureConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal temperature value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("temperature value expose %q: %s", wrapper.temperatureProperty, wrapper.temperatureConverter)
				}
				if !wrapper.environment.Temperature.IsSet() {
					wrapper.environment.Temperature.Set(0)
				}

			case expose.Property == "humidity":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.humidityProperty = expose.Property
				wrapper.humidityConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal humidity value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("humidity value expose %q: %s", wrapper.humidityProperty, wrapper.humidityConverter)
				}
				if !wrapper.environment.Humidity.IsSet() {
					wrapper.environment.Humidity.Set(0)
				}

			case expose.Property == "pressure":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.pressureProperty = expose.Property
				wrapper.pressureConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal pressure value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("pressure value expose %q: %s", wrapper.pressureProperty, wrapper.pressureConverter)
				}
				if !wrapper.environment.Pressure.IsSet() {
					wrapper.environment.Pressure.Set(0)
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperEnvironment) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		if key == "" {
			continue
		}
		switch key {
		case wrapper.temperatureProperty:
			handled = append(handled, key)
			val, err := wrapper.temperatureConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal temperature value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("temperature value %q: %s", wrapper.temperatureProperty, val)
				wrapper.environment.Temperature.Set(val)
			}

		case wrapper.humidityProperty:
			handled = append(handled, key)
			val, err := wrapper.humidityConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal humidity value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("humidity value %q: %s", wrapper.humidityProperty, val)
				wrapper.environment.Humidity.Set(val)
			}

		case wrapper.pressureProperty:
			handled = append(handled, key)
			val, err := wrapper.pressureConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal pressure value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("pressure value %q: %s", wrapper.pressureProperty, val)
				wrapper.environment.Pressure.Set(val)
			}
		}
	}
	return handled
}
