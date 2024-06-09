package zigbee

import (
	"encoding/json"
	"fmt"

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

	dev     *devices.Device
	info    *services.Info
	online  *services.Online
	climate *services.Climate
	battery *services.Battery

	heatingSetpointProperty  string
	heatingSetpointConverter *NumericConverter

	localTemperatureProperty  string
	localTemperatureConverter *NumericConverter

	piHeatingDemandProperty  string
	piHeatingDemandConverter *NumericConverter

	batteryProperty  string
	batteryConverter *NumericConverter

	voltageProperty  string
	voltageConverter *NumericConverter
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
		climate:  services.NewClimate("climate"),
		// battery: created when detected
	}

	dev.dev.AddService(
		dev.info,
		dev.online,
		dev.climate,
	)
	dev.climate.OnAction(dev.handleClimateAction)

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

	var err error
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "climate":
			feature, err := UnmarshalFeature(expose.Data)
			if err != nil {
				dev.log.Errorf("failed to unmarshal climate: %s -- %s", err, expose)
			} else {
				for _, featureExpose := range feature {
					switch featureExpose.Name {
					case "occupied_heating_setpoint", "current_heating_setpoint":
						dev.heatingSetpointProperty = featureExpose.Property
						dev.heatingSetpointConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal heating setpoint: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("heating setpoint expose %q: %s", dev.heatingSetpointProperty, dev.heatingSetpointConverter)
						}
						if dev.heatingSetpointConverter.ValueMin != nil && dev.heatingSetpointConverter.ValueMax != nil && dev.heatingSetpointConverter.ValueStep != nil {
							dev.climate.HeatingSetpoint.SetLimits(*dev.heatingSetpointConverter.ValueMin, *dev.heatingSetpointConverter.ValueMax, *dev.heatingSetpointConverter.ValueStep)
							dev.climate.HeatingSetpoint.Set(*dev.heatingSetpointConverter.ValueMin)
						} else {
							min, _, _ := dev.climate.HeatingSetpoint.GetLimits()
							dev.climate.HeatingSetpoint.Set(min)
						}

					case "local_temperature":
						dev.localTemperatureProperty = featureExpose.Property
						dev.localTemperatureConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal local temperature: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("local temperature expose %q: %s", dev.localTemperatureProperty, dev.localTemperatureConverter)
						}
						if dev.localTemperatureConverter.ValueMin != nil && dev.localTemperatureConverter.ValueMax != nil && dev.localTemperatureConverter.ValueStep != nil {
							dev.climate.LocalTemperature.SetLimits(*dev.localTemperatureConverter.ValueMin, *dev.localTemperatureConverter.ValueMax, *dev.localTemperatureConverter.ValueStep)
							dev.climate.LocalTemperature.Set(*dev.localTemperatureConverter.ValueMin)
						} else {
							min, _, _ := dev.climate.LocalTemperature.GetLimits()
							dev.climate.LocalTemperature.Set(min)
						}

					case "pi_heating_demand":
						dev.piHeatingDemandProperty = featureExpose.Property
						dev.piHeatingDemandConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal pi heating demand: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("pi heating demand expose %q: %s", dev.piHeatingDemandProperty, dev.piHeatingDemandConverter)
						}
						if dev.piHeatingDemandConverter.ValueMin != nil && dev.piHeatingDemandConverter.ValueMax != nil && dev.piHeatingDemandConverter.ValueStep != nil {
							dev.climate.PIHeatingDemand.SetLimits(int64(*dev.piHeatingDemandConverter.ValueMin), int64(*dev.piHeatingDemandConverter.ValueMax), uint64(*dev.piHeatingDemandConverter.ValueStep))
							dev.climate.PIHeatingDemand.Set(int64(*dev.piHeatingDemandConverter.ValueMin))
						} else {
							min, _, _ := dev.climate.PIHeatingDemand.GetLimits()
							dev.climate.PIHeatingDemand.Set(min)
						}

					default:
						dev.log.Warnf("unsupported climate expose %q: %s", featureExpose.Name, featureExpose)
					}
				}
			}

		case "numeric":
			switch {
			case expose.Property == "battery" && expose.Category == "diagnostic":
				dev.batteryProperty = expose.Property
				dev.batteryConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					dev.log.Errorf("failed to unmarshal battery level value: %s -- %s", err, expose)
				} else {
					dev.log.Debugf("battery level value expose %q: %s", dev.batteryProperty, dev.batteryConverter)
				}
				if dev.battery == nil {
					dev.battery = services.NewBattery("")
					dev.dev.AddService(dev.battery)
				}
				dev.battery.Level.Set(0)

			case expose.Property == "voltage" && expose.Category == "diagnostic":
				dev.voltageProperty = expose.Property
				dev.voltageConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					dev.log.Errorf("failed to unmarshal battery voltage value: %s -- %s", err, expose)
				} else {
					dev.log.Debugf("battery voltage value expose %q: %s", dev.voltageProperty, dev.voltageConverter)
				}
				if dev.battery == nil {
					dev.battery = services.NewBattery("")
					dev.dev.AddService(dev.battery)
				}
				if dev.voltageConverter.ValueMin != nil && dev.voltageConverter.ValueMax != nil && dev.voltageConverter.ValueStep != nil {
					dev.battery.Voltage.SetLimits(*dev.voltageConverter.ValueMin, *dev.voltageConverter.ValueMax, *dev.voltageConverter.ValueStep)
					dev.battery.Voltage.Set(*dev.voltageConverter.ValueMin)
				} else {
					dev.battery.Voltage.Set(0)
				}
			}

		default:
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

	for key, value := range state.Values {
		switch key {
		case dev.heatingSetpointProperty:
			val, err := dev.heatingSetpointConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal heating setpoint value %q: %s", value, err)
			} else {
				dev.log.Debugf("heating setpoint value %q: %v", dev.heatingSetpointProperty, val)
				dev.climate.HeatingSetpoint.Set(val)
			}

		case dev.localTemperatureProperty:
			val, err := dev.localTemperatureConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal local temperature value %q: %s", value, err)
			} else {
				dev.log.Debugf("local temperature value %q: %v", dev.localTemperatureProperty, val)
				dev.climate.LocalTemperature.Set(val)
			}

		case dev.piHeatingDemandProperty:
			val, err := dev.piHeatingDemandConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal pi heating demand value %q: %s", value, err)
			} else {
				piHeatingDemand := int64(val)
				dev.log.Debugf("pi heating demand value %q: %f -> %d", dev.piHeatingDemandProperty, val, piHeatingDemand)
				dev.climate.PIHeatingDemand.Set(piHeatingDemand)
			}

		case dev.batteryProperty:
			val, err := dev.batteryConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal battery level value %q: %s", value, err)
			} else {
				level := int64(val)
				dev.log.Debugf("battery level value %q: %f -> %d", dev.batteryProperty, val, level)
				dev.battery.Level.Set(level)
			}

		case dev.voltageProperty:
			val, err := dev.voltageConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal battery voltage value %q: %s", value, err)
			} else {
				dev.log.Debugf("battery voltage value %q: %v", dev.voltageProperty, val)
				dev.battery.Voltage.Set(val)
			}

		default:
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

func (dev *ZigbeeClimate) handleClimateAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	dev.log.Debugf("handling request: %s", request)
	if dev.requests != nil {
		reqjson := make(map[string]json.RawMessage)
		for _, val := range request.Values {
			switch val.Id {
			case dev.climate.HeatingSetpoint.ID():
				if dev.heatingSetpointConverter != nil {
					valjson, err := dev.heatingSetpointConverter.MarshalValue(val.GetFloat().GetValue())
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[dev.heatingSetpointProperty] = valjson
				} else {
					dev.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			default:
				dev.log.Errorf("unsupported request value: %s", val)
				return fmt.Errorf("unsupported request value: %s", val)
			}
		}

		dev.log.Debugf("handling request: %s", reqjson)
		if len(reqjson) > 0 {
			data, err := json.Marshal(reqjson)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal reqjson: %s --- %s", reqjson, err))
			}
			dev.requests(ZigbeeRequest{Topic: dev.friendlyName + "/set", Payload: data})
		}
	}
	return nil
}
