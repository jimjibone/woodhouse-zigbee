package zigbee

import (
	"encoding/json"
	"fmt"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperClimate)(nil)

type WrapperClimate struct {
	log      *log.Context
	requests func(payload []byte)

	climate *services.Climate

	heatingSetpointProperty  string
	heatingSetpointConverter *NumericConverter

	localTemperatureProperty  string
	localTemperatureConverter *NumericConverter

	piHeatingDemandProperty  string
	piHeatingDemandConverter *NumericConverter
}

func SupportsClimate(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "climate":
			// This is the feature we need.
			return true
		}
	}
	return false
}

func NewWrapperClimate(log *log.Context, dev *devices.Device, requests func(payload []byte)) *WrapperClimate {
	wrapper := &WrapperClimate{
		log:      log,
		climate:  services.NewClimate(""),
		requests: requests,
	}
	wrapper.climate.OnAction(wrapper.handleAction)
	dev.AddService(wrapper.climate)
	return wrapper
}

func (wrapper *WrapperClimate) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "climate":
			handled = append(handled, HandledExpose{expose.Type, expose.Property})
			feature, err := UnmarshalFeature(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal climate: %s -- %s", err, expose)
			} else {
				for _, featureExpose := range feature {
					switch featureExpose.Name {
					case "occupied_heating_setpoint", "current_heating_setpoint":
						wrapper.heatingSetpointProperty = featureExpose.Property
						wrapper.heatingSetpointConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal heating setpoint: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("heating setpoint expose %q: %s", wrapper.heatingSetpointProperty, wrapper.heatingSetpointConverter)
						}
						if wrapper.heatingSetpointConverter.ValueMin != nil && wrapper.heatingSetpointConverter.ValueMax != nil && wrapper.heatingSetpointConverter.ValueStep != nil {
							wrapper.climate.HeatingSetpoint.SetLimits(*wrapper.heatingSetpointConverter.ValueMin, *wrapper.heatingSetpointConverter.ValueMax, *wrapper.heatingSetpointConverter.ValueStep)
							wrapper.climate.HeatingSetpoint.Set(*wrapper.heatingSetpointConverter.ValueMin)
						} else {
							min, _, _ := wrapper.climate.HeatingSetpoint.GetLimits()
							wrapper.climate.HeatingSetpoint.Set(min)
						}

					case "local_temperature":
						wrapper.localTemperatureProperty = featureExpose.Property
						wrapper.localTemperatureConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal local temperature: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("local temperature expose %q: %s", wrapper.localTemperatureProperty, wrapper.localTemperatureConverter)
						}
						if wrapper.localTemperatureConverter.ValueMin != nil && wrapper.localTemperatureConverter.ValueMax != nil && wrapper.localTemperatureConverter.ValueStep != nil {
							wrapper.climate.LocalTemperature.SetLimits(*wrapper.localTemperatureConverter.ValueMin, *wrapper.localTemperatureConverter.ValueMax, *wrapper.localTemperatureConverter.ValueStep)
							wrapper.climate.LocalTemperature.Set(*wrapper.localTemperatureConverter.ValueMin)
						} else {
							min, _, _ := wrapper.climate.LocalTemperature.GetLimits()
							wrapper.climate.LocalTemperature.Set(min)
						}

					case "pi_heating_demand":
						wrapper.piHeatingDemandProperty = featureExpose.Property
						wrapper.piHeatingDemandConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal pi heating demand: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("pi heating demand expose %q: %s", wrapper.piHeatingDemandProperty, wrapper.piHeatingDemandConverter)
						}
						if wrapper.piHeatingDemandConverter.ValueMin != nil && wrapper.piHeatingDemandConverter.ValueMax != nil && wrapper.piHeatingDemandConverter.ValueStep != nil {
							wrapper.climate.PIHeatingDemand.SetLimits(int64(*wrapper.piHeatingDemandConverter.ValueMin), int64(*wrapper.piHeatingDemandConverter.ValueMax), uint64(*wrapper.piHeatingDemandConverter.ValueStep))
							wrapper.climate.PIHeatingDemand.Set(int64(*wrapper.piHeatingDemandConverter.ValueMin))
						} else {
							min, _, _ := wrapper.climate.PIHeatingDemand.GetLimits()
							wrapper.climate.PIHeatingDemand.Set(min)
						}

					default:
						wrapper.log.Warnf("unsupported climate expose %q: %s", featureExpose.Name, featureExpose)
					}
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperClimate) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		switch key {
		case wrapper.heatingSetpointProperty:
			handled = append(handled, key)
			val, err := wrapper.heatingSetpointConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal heating setpoint value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("heating setpoint value %q: %v", wrapper.heatingSetpointProperty, val)
				wrapper.climate.HeatingSetpoint.Set(val)
			}

		case wrapper.localTemperatureProperty:
			handled = append(handled, key)
			val, err := wrapper.localTemperatureConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal local temperature value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("local temperature value %q: %v", wrapper.localTemperatureProperty, val)
				wrapper.climate.LocalTemperature.Set(val)
			}

		case wrapper.piHeatingDemandProperty:
			handled = append(handled, key)
			val, err := wrapper.piHeatingDemandConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal pi heating demand value %q: %s", value, err)
			} else {
				piHeatingDemand := int64(val)
				wrapper.log.Debugf("pi heating demand value %q: %f -> %d", wrapper.piHeatingDemandProperty, val, piHeatingDemand)
				wrapper.climate.PIHeatingDemand.Set(piHeatingDemand)
			}
		}
	}
	return handled
}

func (wrapper *WrapperClimate) handleAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	wrapper.log.Debugf("handling request: %s", request)
	if wrapper.requests != nil {
		reqjson := make(map[string]json.RawMessage)
		for _, val := range request.Values {
			switch val.Id {
			case wrapper.climate.HeatingSetpoint.ID():
				if wrapper.heatingSetpointConverter != nil {
					valjson, err := wrapper.heatingSetpointConverter.MarshalValue(val.GetFloat().GetValue())
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[wrapper.heatingSetpointProperty] = valjson
				} else {
					wrapper.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			default:
				wrapper.log.Errorf("unsupported request value: %s", val)
				return fmt.Errorf("unsupported request value: %s", val)
			}
		}

		wrapper.log.Debugf("handling request: %s", reqjson)
		if len(reqjson) > 0 {
			data, err := json.Marshal(reqjson)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal reqjson: %s --- %s", reqjson, err))
			}
			wrapper.requests(data)
		}
	}
	return nil
}
