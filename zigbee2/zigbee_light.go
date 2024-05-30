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
	log    *log.Context
	client *wh.Client
	added  bool

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

	ignoreColor       bool
	colorProperty     string
	colorHueProperty  string
	colorSatProperty  string
	colorHueConverter *NumericConverter
	colorSatConverter *NumericConverter
	colorXProperty    string
	colorYProperty    string
	colorXConverter   *NumericConverter
	colorYConverter   *NumericConverter
}

func NewZigbeeLight(info DeviceInfo, client *wh.Client, baseUrl string, requests func(ZigbeeRequest)) *ZigbeeLight {
	dev := &ZigbeeLight{
		log:       log.NewContext(log.DefaultLogger, info.IEEEAddress, log.DebugLevel),
		client:    client,
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
						dev.lightbulb.On.Set(false)

					case "brightness":
						dev.briProperty = featureExpose.Property
						dev.briConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal light brightness: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("brightness expose %q: %s", dev.briProperty, dev.briConverter)
						}
						dev.lightbulb.Brightness.Set(0)

					case "color_temp":
						dev.ctProperty = featureExpose.Property
						dev.ctConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal light color_temp: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("color_temp expose %q: %s", dev.ctProperty, dev.ctConverter)
						}
						if dev.ctConverter.ValueMin != nil && dev.ctConverter.ValueMax != nil && dev.ctConverter.ValueStep != nil {
							dev.lightbulb.ColorTemp.SetLimits(int64(*dev.ctConverter.ValueMin), int64(*dev.ctConverter.ValueMax), uint64(*dev.ctConverter.ValueStep))
							dev.lightbulb.ColorTemp.Set(int64(*dev.ctConverter.ValueMin))
						} else {
							min, _, _ := dev.lightbulb.ColorTemp.GetLimits()
							dev.lightbulb.ColorTemp.Set(min)
						}

					case "color_xy":
						dev.colorProperty = featureExpose.Property
						colorFeatures, err := UnmarshalFeature(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal light color_xy: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("color_xy expose %q: %s", dev.colorProperty, colorFeatures)

							for _, colorFeature := range colorFeatures {
								switch colorFeature.Name {
								case "x":
									dev.colorXProperty = colorFeature.Property
									dev.colorXConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										dev.log.Errorf("failed to unmarshal light color_xy.x: %s -- %s", err, expose)
									} else {
										dev.log.Debugf("color_xy.x expose %q: %s", dev.colorXProperty, dev.colorXConverter)
									}

								case "y":
									dev.colorYProperty = colorFeature.Property
									dev.colorYConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										dev.log.Errorf("failed to unmarshal light color_xy.y: %s -- %s", err, expose)
									} else {
										dev.log.Debugf("color_xy.y expose %q: %s", dev.colorYProperty, dev.colorYConverter)
									}

								default:
									dev.log.Errorf("unsupported light color_xy feature %q: %s", colorFeature.Name, colorFeature)
								}
							}
						}

					case "color_hs":
						dev.colorProperty = featureExpose.Property
						colorFeatures, err := UnmarshalFeature(featureExpose.Data)
						if err != nil {
							dev.log.Errorf("failed to unmarshal light color_hs: %s -- %s", err, expose)
						} else {
							dev.log.Debugf("color_hs expose %q: %s", dev.colorProperty, colorFeatures)

							for _, colorFeature := range colorFeatures {
								switch colorFeature.Name {
								case "hue":
									dev.colorHueProperty = colorFeature.Property
									dev.colorHueConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										dev.log.Errorf("failed to unmarshal light color_hs.hue: %s -- %s", err, expose)
									} else {
										dev.log.Debugf("color_hs.hue expose %q: %s", dev.colorHueProperty, dev.colorHueConverter)
									}

								case "saturation":
									dev.colorSatProperty = colorFeature.Property
									dev.colorSatConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										dev.log.Errorf("failed to unmarshal light color_hs.sat: %s -- %s", err, expose)
									} else {
										dev.log.Debugf("color_hs.sat expose %q: %s", dev.colorSatProperty, dev.colorSatConverter)
									}

								default:
									dev.log.Errorf("unsupported light color_hs feature %q: %s", colorFeature.Name, colorFeature)
								}
							}
						}

					case "color_mode":
						// ignore

					default:
						dev.log.Errorf("unsupported light expose %q: %s", featureExpose.Name, featureExpose)
					}
				}
			}

		default:
			dev.log.Errorf("unsupported expose type %q: %s", expose.Type, expose)
		}

		if dev.colorHueProperty == "" && dev.colorSatProperty == "" {
			// If the info reports x/y support but not hue/sat then don't allow
			// color control.
			dev.ignoreColor = true
		} else {
			// Otherwise set default values.
			dev.lightbulb.Color.Set(0, 0, 0, 0)
		}
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

		case dev.colorProperty:
			if !dev.ignoreColor {
				var vals map[string]json.RawMessage
				err := json.Unmarshal(value, &vals)
				if err != nil {
					dev.log.Errorf("failed to unmarshal color value %q: %s", value, err)
				} else {
					for colorKey, colorValue := range vals {
						hue, sat, x, y := dev.lightbulb.Color.Get()
						switch colorKey {
						case dev.colorXProperty:
							val, err := dev.colorXConverter.UnmarshalValue(colorValue)
							if err != nil {
								dev.log.Errorf("failed to unmarshal color.x value %q: %s", colorValue, err)
							} else {
								dev.log.Debugf("color.x value %q: %f", dev.colorXProperty, val)
								x = val
							}

						case dev.colorYProperty:
							val, err := dev.colorYConverter.UnmarshalValue(colorValue)
							if err != nil {
								dev.log.Errorf("failed to unmarshal color.y value %q: %s", colorValue, err)
							} else {
								dev.log.Debugf("color.y value %q: %f", dev.colorYProperty, val)
								y = val
							}

						case dev.colorHueProperty:
							val, err := dev.colorHueConverter.UnmarshalValue(colorValue)
							if err != nil {
								dev.log.Errorf("failed to unmarshal color.hue value %q: %s", colorValue, err)
							} else {
								dev.log.Debugf("color.hue value %q: %f", dev.colorHueProperty, val)
								hue = val
							}

						case dev.colorSatProperty:
							val, err := dev.colorSatConverter.UnmarshalValue(colorValue)
							if err != nil {
								dev.log.Errorf("failed to unmarshal color.sat value %q: %s", colorValue, err)
							} else {
								dev.log.Debugf("color.sat value %q: %f", dev.colorSatProperty, val)
								sat = val
							}

						default:
							dev.log.Errorf("unsupported color property %q: %s", colorKey, colorValue)
						}
						dev.lightbulb.Color.Set(hue, sat, x, y)
					}
				}
			}

		case "color_mode":
			// ignore

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

			case dev.lightbulb.Color.ID():
				if val.GetColor().HueSat != nil {
					if dev.colorHueConverter != nil && dev.colorSatConverter != nil {
						huejson, err := dev.colorHueConverter.MarshalValue(val.GetColor().GetHueSat().Hue)
						if err != nil {
							return fmt.Errorf("marshal hue %s: %s", val, err)
						}
						satjson, err := dev.colorSatConverter.MarshalValue(val.GetColor().GetHueSat().Sat)
						if err != nil {
							return fmt.Errorf("marshal sat %s: %s", val, err)
						}
						huesat := map[string]json.RawMessage{
							dev.colorHueProperty: huejson,
							dev.colorSatProperty: satjson,
						}
						huesatjson, err := json.Marshal(huesat)
						if err != nil {
							return fmt.Errorf("marshal feature %s: %s", val, err)
						}
						reqjson[dev.colorProperty] = huesatjson
					} else {
						dev.log.Errorf("no converter for %s", val)
						return fmt.Errorf("no converter for %s", val)
					}
				} else if val.GetColor().Xy != nil {
					if dev.colorXConverter != nil && dev.colorYConverter != nil {
						xjson, err := dev.colorXConverter.MarshalValue(val.GetColor().GetXy().X)
						if err != nil {
							return fmt.Errorf("marshal x %s: %s", val, err)
						}
						yjson, err := dev.colorYConverter.MarshalValue(val.GetColor().GetXy().Y)
						if err != nil {
							return fmt.Errorf("marshal y %s: %s", val, err)
						}
						xy := map[string]json.RawMessage{
							dev.colorXProperty: xjson,
							dev.colorYProperty: yjson,
						}
						xyjson, err := json.Marshal(xy)
						if err != nil {
							return fmt.Errorf("marshal feature %s: %s", val, err)
						}
						reqjson[dev.colorProperty] = xyjson
					} else {
						dev.log.Errorf("no converter for %s", val)
						return fmt.Errorf("no converter for %s", val)
					}
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
