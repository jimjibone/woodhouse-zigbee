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

type ZigbeeLight struct {
	log *log.Context

	baseUrl      string
	friendlyName string
	requests     func(ZigbeeRequest)

	dev       *devices.Device
	info      *services.Info
	online    *services.Online
	lightbulb *services.Lightbulb

	onProperty  string
	onConverter *BinaryConverter

	briProperty  string
	briConverter *NumericConverter

	ctProperty  string
	ctConverter *NumericConverter

	// colorProperty  string
	// colorConverter *NumericConverter
}

func NewZigbeeLight(info DeviceInfo, client *wh.Client, baseUrl string, requests func(ZigbeeRequest)) *ZigbeeLight {
	dev := &ZigbeeLight{
		log:       log.NewContext(log.DefaultLogger, info.IEEEAddress, log.DebugLevel),
		baseUrl:   baseUrl,
		requests:  requests,
		dev:       devices.NewDevice(info.IEEEAddress, clientsapi.Device_LIGHTBULB),
		info:      services.NewInfo(),
		online:    services.NewOnline(),
		lightbulb: services.NewLightbulb("lightbulb"),
	}

	dev.dev.AddService(
		dev.info,
		dev.online,
		dev.lightbulb,
	)
	dev.lightbulb.OnAction(dev.handleLightAction)

	dev.log.Infof("created lightbulb")

	dev.UpdateInfo(info)

	err := client.AddDevice(dev.dev)
	if err != nil {
		dev.log.Fatalf("failed to add device: %s", err)
	}

	return dev
}

func (dev *ZigbeeLight) Device() *devices.Device { return dev.dev }
func (dev *ZigbeeLight) Name() string            { return dev.friendlyName }

func (dev *ZigbeeLight) UpdateInfo(info DeviceInfo) {
	dev.friendlyName = info.FriendlyName
	dev.info.Name.Set(info.FriendlyName)
	dev.info.Model.Set(info.ModelID)
	dev.info.Manufacturer.Set(info.Manufacturer)
	dev.info.SerialNumber.Set(info.IEEEAddress)
	dev.info.FirmwareVersion.Set(info.SoftwareBuildID)
	dev.info.WebUrl.Set(fmt.Sprintf("http://%s/#/device/%s/info", dev.baseUrl, info.IEEEAddress))

	dev.log.Debugf("info: %v", info)

	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "light":
			feature, err := UnmarshalFeature(expose.Data)
			if err != nil {
				dev.log.Errorf("failed to unmarshal light: %s -- %s", err, expose)
			} else {
				for _, featureExpose := range feature {
					switch featureExpose.Name {
					case "state":
						dev.onProperty = featureExpose.Property
						dev.onConverter, err = UnmarshalBinary(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal light state: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("on expose %q: %s", dev.onProperty, dev.onConverter)
						}

					case "brightness":
						dev.briProperty = featureExpose.Property
						dev.briConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal light brightness: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("brightness expose %q: %s", dev.briProperty, dev.briConverter)
						}

					case "color_temp":
						dev.ctProperty = featureExpose.Property
						dev.ctConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal light color_temp: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("color_temp expose %q: %s", dev.ctProperty, dev.ctConverter)
						}

					// case "color_hs":
					// 	dev.colorProperty = featureExpose.Property
					// 	dev.colorConverter, err = UnmarshalNumeric(featureExpose.Data)
					// 	if err != nil {
					// 		dev.log.Errorf("failed to unmarshal light brightness: %s -- %s", err, expose)
					// 	} else {
					// 		dev.log.Debugf("brightness expose %q: %s", dev.colorProperty, dev.colorConverter)
					// 	}

					default:
						dev.log.Errorf("unsupported light expose %q: %s", featureExpose.Name, featureExpose)
					}
				}
			}

		default:
			dev.log.Errorf("unsupported expose type %q: %s", expose.Type, expose)
		}

		// switch expose.Type {
		// case "binary":
		// 	conv, err := converters.NewBinary(expose.Data)
		// 	if err != nil {
		// 		dev.log.Errorf("binary %q: %w", expose.Property, err)
		// 	} else {
		// 		dev.log.Debugf("binary %s: %s", expose.Property, conv)
		// 		dev.converters[expose.Property] = conv
		// 	}

		// case "numeric":
		// 	conv, err := converters.NewNumeric(expose.Data)
		// 	if err != nil {
		// 		dev.log.Errorf("numeric %q: %w", expose.Property, err)
		// 	} else {
		// 		dev.log.Debugf("numeric %s: %s", expose.Property, conv)
		// 		dev.converters[expose.Property] = conv
		// 	}

		// case "enum":
		// 	conv, err := converters.NewEnum(expose.Data)
		// 	if err != nil {
		// 		dev.log.Errorf("enum %q: %w", expose.Property, err)
		// 	} else {
		// 		dev.log.Debugf("enum %s: %s", expose.Property, conv)
		// 		dev.converters[expose.Property] = conv
		// 	}

		// case "text":
		// 	conv, err := converters.NewText(expose.Data)
		// 	if err != nil {
		// 		dev.log.Errorf("text %q: %w", expose.Property, err)
		// 	} else {
		// 		dev.log.Debugf("text %s: %s", expose.Property, conv)
		// 		dev.converters[expose.Property] = conv
		// 	}

		// case "light", "climate", "switch", "fan", "cover", "lock":
		// 	convs, err := converters.NewFeature(expose.Data)
		// 	if err != nil {
		// 		dev.log.Errorf("feature %q: %w", expose.Type, err)
		// 	} else {
		// 		for property, conv := range convs {
		// 			dev.log.Debugf("feature %s.%s: %s", expose.Type, property, conv)
		// 			dev.converters[property] = conv
		// 		}
		// 	}

		// default:
		// 	dev.log.Errorf("unknown expose type %q", expose.Type)
		// }
	}
}

func (dev *ZigbeeLight) UpdateOnline(online bool) {
	dev.online.Online.Set(online)
}

func (dev *ZigbeeLight) UpdateState(state DeviceState) {
	dev.log.Debugf("state: %v", state)

	dev.online.LastSeen.Set(state.LastSeen)

	for key, value := range state.Values {
		switch key {
		case dev.onProperty:
			val, err := dev.onConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal on value %q: %s", value, err)
			} else {
				dev.log.Debugf("on value %q: %v", dev.onProperty, val)
				dev.lightbulb.On.Set(val)
			}

		case dev.briProperty:
			val, err := dev.briConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal brightness value %q: %s", value, err)
			} else {
				var bri int64
				if dev.briConverter.ValueMax != nil {
					bri = int64(val / *dev.briConverter.ValueMax * 100.0)
				} else {
					bri = int64(val)
				}
				dev.log.Debugf("brightness value %q: %f -> %d", dev.briProperty, val, bri)
				dev.lightbulb.Brightness.Set(bri)
			}

		case dev.ctProperty:
			val, err := dev.ctConverter.UnmarshalValue(value)
			if err != nil {
				dev.log.Errorf("failed to unmarshal color_temp value %q: %s", value, err)
			} else {
				ct := int64(val)
				dev.log.Debugf("color_temp value %q: %f -> %d", dev.ctProperty, val, ct)
				dev.lightbulb.ColorTemp.Set(ct)
			}

		// case "color_mode":
		// case "color_temp":
		// case "color":
		default:
			dev.log.Errorf("unsupported state property %q: %s", key, value)
		}
	}

	// 	changed := false
	// 	if !zd.lastSeen.After(state.LastSeen) {
	// 		changed = true
	// 		zd.lastSeen = state.LastSeen
	// 	}

	// 	for name, value := range state.Values {
	// 		if converter, found := zd.converters[name]; found {
	// 			// Use the converter to convert this state value to a woodhouse value.
	// 			next, err := converter.Unmarshal(value)
	// 			if err != nil {
	// 				log.Printf("ERROR: device %s failed to convert value %q with %s: %s", zd.id, name, value, err)
	// 			} else {
	// 				if prev, found := zd.values[name]; found {
	// 					log.Printf("device %s updated value %q: %v --> %v (converted)", zd.id, name, prev, next)
	// 				} else {
	// 					log.Printf("device %s new value %q: %v (converted)", zd.id, name, next)
	// 				}
	// 				changed = true
	// 				zd.values[name] = next
	// 			}
	// 		} else {
	// 			// Do a direct conversion.
	// 			var val interface{}
	// 			err := json.Unmarshal(value, &val)
	// 			if err != nil {
	// 				log.Printf("ERROR: device %s failed to unmarshal value %q with %s", zd.id, name, value)
	// 			} else {
	// 				var next *api.DeviceValue
	// 				switch v := val.(type) {
	// 				case bool:
	// 					next = converters.ConvertBool(v)

	// 				case float64:
	// 					next = converters.ConvertNumber(v)

	// 				case string:
	// 					next = converters.ConvertText(v)

	// 				case nil:
	// 					// Ignore.

	// 				default:
	// 					switch name {
	// 					case "update":
	// 						// Ignore.

	// 					default:
	// 						log.Printf("ERROR: device %s failed to convert value %q with %+v: no converter", zd.id, name, val)
	// 					}
	// 				}
	// 				if next != nil {
	// 					if prev, found := zd.values[name]; found {
	// 						log.Printf("device %s updated value %q: %v --> %v", zd.id, name, prev, next)
	// 					} else {
	// 						log.Printf("device %s new value %q: %v", zd.id, name, next)
	// 					}
	// 					changed = true
	// 					zd.values[name] = next
	// 				}
	// 			}
	// 		}
	// 	}

	// 	if changed && zd.comms != nil {
	// 		msg := &api.DeviceState{
	// 			DeviceId:   zd.ID(),
	// 			Online:     zd.online,
	// 			LastSeen:   apitools.TimeToTimestamp(zd.lastSeen),
	// 			FullUpdate: true,
	// 			Values:     []*api.DeviceValue{},
	// 		}
	// 		for name, val := range zd.values {
	// 			val.Name = name
	// 			msg.Values = append(msg.Values, proto.Clone(val).(*api.DeviceValue))
	// 		}
	// 		err := zd.comms.SendState(msg)
	// 		if err != nil {
	// 			log.Printf("ERROR: device %s failed to send state: %s", zd.id, err)
	// 		}
	// 	}
}

func (dev *ZigbeeLight) handleLightAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	dev.log.Debugf("handling request: %s", request)
	if dev.requests != nil {
		reqjson := make(map[string]json.RawMessage)
		for _, val := range request.Values {
			switch val.Id {
			case dev.lightbulb.On.ID():
				if dev.onConverter != nil {
					valjson, err := dev.onConverter.MarshalValue(val.GetBool().GetValue())
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[dev.onProperty] = valjson
				} else {
					dev.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			case dev.lightbulb.Brightness.ID():
				if dev.briConverter != nil {
					var bri float64
					if dev.briConverter.ValueMax != nil {
						bri = float64(val.GetInt().GetValue()) / 100.0 * *dev.briConverter.ValueMax
					} else {
						bri = float64(val.GetInt().GetValue())
					}
					valjson, err := dev.briConverter.MarshalValue(bri)
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[dev.briProperty] = valjson
				} else {
					dev.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			case dev.lightbulb.ColorTemp.ID():
				if dev.ctConverter != nil {
					ct := float64(val.GetInt().GetValue())
					valjson, err := dev.ctConverter.MarshalValue(ct)
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[dev.ctProperty] = valjson
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
