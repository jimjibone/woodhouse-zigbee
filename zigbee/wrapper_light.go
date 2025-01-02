package zigbee

import (
	"encoding/json"
	"fmt"
	"math"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperLight)(nil)

type WrapperLight struct {
	log      *log.Context
	requests func(payload []byte)

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

	transitionProperty  string
	transitionConverter *NumericConverter
}

func SupportsLight(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "light":
			// This is the feature we need.
			return true
		}
	}
	return false
}

func NewWrapperLight(log *log.Context, dev *devices.Device, requests func(payload []byte)) *WrapperLight {
	wrapper := &WrapperLight{
		log:       log,
		lightbulb: services.NewLightbulb(""),
		requests:  requests,
	}
	wrapper.lightbulb.OnAction(wrapper.handleAction)
	dev.AddService(wrapper.lightbulb)
	return wrapper
}

func (wrapper *WrapperLight) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "light":
			handled = append(handled, HandledExpose{Type: expose.Type, Property: expose.Property})
			feature, err := UnmarshalFeature(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal light: %s -- %s", err, expose)
			} else {
				for _, featureExpose := range feature {
					switch featureExpose.Name {
					case "state":
						wrapper.onProperty = featureExpose.Property
						wrapper.onConverter, err = UnmarshalBinary(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal light state: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("on expose %q: %s", wrapper.onProperty, wrapper.onConverter)
						}
						if !wrapper.lightbulb.On.IsSet() {
							wrapper.lightbulb.On.Set(false)
						}

					case "brightness":
						wrapper.briProperty = featureExpose.Property
						wrapper.briConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal light brightness: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("brightness expose %q: %s", wrapper.briProperty, wrapper.briConverter)
						}
						if !wrapper.lightbulb.Brightness.IsSet() {
							wrapper.lightbulb.Brightness.Set(0)
						}

					case "color_temp":
						wrapper.ctProperty = featureExpose.Property
						wrapper.ctConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal light color_temp: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("color_temp expose %q: %s", wrapper.ctProperty, wrapper.ctConverter)
						}
						if wrapper.ctConverter.ValueMin != nil && wrapper.ctConverter.ValueMax != nil && wrapper.ctConverter.ValueStep != nil {
							wrapper.lightbulb.ColorTemp.SetLimits(int64(*wrapper.ctConverter.ValueMin), int64(*wrapper.ctConverter.ValueMax), uint64(*wrapper.ctConverter.ValueStep))
							if !wrapper.lightbulb.ColorTemp.IsSet() {
								wrapper.lightbulb.ColorTemp.Set(int64(*wrapper.ctConverter.ValueMin))
							}
						} else {
							min, _, _ := wrapper.lightbulb.ColorTemp.GetLimits()
							if !wrapper.lightbulb.ColorTemp.IsSet() {
								wrapper.lightbulb.ColorTemp.Set(min)
							}
						}

					case "color_xy":
						wrapper.colorProperty = featureExpose.Property
						colorFeatures, err := UnmarshalFeature(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal light color_xy: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("color_xy expose %q: %s", wrapper.colorProperty, colorFeatures)

							for _, colorFeature := range colorFeatures {
								switch colorFeature.Name {
								case "x":
									wrapper.colorXProperty = colorFeature.Property
									wrapper.colorXConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										wrapper.log.Errorf("failed to unmarshal light color_xy.x: %s -- %s", err, expose)
									} else {
										wrapper.log.Debugf("color_xy.x expose %q: %s", wrapper.colorXProperty, wrapper.colorXConverter)
									}

								case "y":
									wrapper.colorYProperty = colorFeature.Property
									wrapper.colorYConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										wrapper.log.Errorf("failed to unmarshal light color_xy.y: %s -- %s", err, expose)
									} else {
										wrapper.log.Debugf("color_xy.y expose %q: %s", wrapper.colorYProperty, wrapper.colorYConverter)
									}

								default:
									wrapper.log.Errorf("unsupported light color_xy feature %q: %s", colorFeature.Name, colorFeature)
								}
							}
						}

					case "color_hs":
						wrapper.colorProperty = featureExpose.Property
						colorFeatures, err := UnmarshalFeature(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal light color_hs: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("color_hs expose %q: %s", wrapper.colorProperty, colorFeatures)

							for _, colorFeature := range colorFeatures {
								switch colorFeature.Name {
								case "hue":
									wrapper.colorHueProperty = colorFeature.Property
									wrapper.colorHueConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										wrapper.log.Errorf("failed to unmarshal light color_hs.hue: %s -- %s", err, expose)
									} else {
										wrapper.log.Debugf("color_hs.hue expose %q: %s", wrapper.colorHueProperty, wrapper.colorHueConverter)
									}

								case "saturation":
									wrapper.colorSatProperty = colorFeature.Property
									wrapper.colorSatConverter, err = UnmarshalNumeric(colorFeature.Data)
									if err != nil {
										wrapper.log.Errorf("failed to unmarshal light color_hs.sat: %s -- %s", err, expose)
									} else {
										wrapper.log.Debugf("color_hs.sat expose %q: %s", wrapper.colorSatProperty, wrapper.colorSatConverter)
									}

								default:
									wrapper.log.Errorf("unsupported light color_hs feature %q: %s", colorFeature.Name, colorFeature)
								}
							}
						}

					case "color_mode":
						// ignore

					default:
						wrapper.log.Warnf("unsupported light expose %q: %s", featureExpose.Name, featureExpose)
					}
				}
			}
		}
	}

	if wrapper.colorHueProperty == "" && wrapper.colorSatProperty == "" {
		// If the info reports x/y support but not hue/sat then don't allow
		// color control.
		wrapper.ignoreColor = true
	} else {
		// Otherwise set default values.
		if !wrapper.lightbulb.Color.IsSet() {
			wrapper.lightbulb.Color.Set(0, 0, 0, 0)
		}
	}

	var err error
	for _, option := range info.Definition.Options {
		switch {
		case option.Name == "transition" && option.Type == "numeric":
			handled = append(handled, HandledExpose{option.Type, option.Property})
			wrapper.transitionProperty = option.Property
			wrapper.transitionConverter, err = UnmarshalNumeric(option.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal transition option: %s -- %s", err, option)
			} else {
				wrapper.log.Debugf("transition value option %q: %s", wrapper.transitionProperty, wrapper.transitionConverter)
			}
			wrapper.lightbulb.Transition.Set(0)
		}
	}

	return handled
}

func (wrapper *WrapperLight) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		switch key {
		case wrapper.onProperty:
			handled = append(handled, key)
			val, err := wrapper.onConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal on value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("on value %q: %v", wrapper.onProperty, val)
				wrapper.lightbulb.On.Set(val)
			}

		case wrapper.briProperty:
			handled = append(handled, key)
			val, err := wrapper.briConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal brightness value %q: %s", value, err)
			} else {
				var bri int64
				if wrapper.briConverter.ValueMax != nil {
					bri = int64(math.Ceil(val / *wrapper.briConverter.ValueMax * 100.0))
				} else {
					bri = int64(val)
				}
				wrapper.log.Debugf("brightness value %q: %f -> %d", wrapper.briProperty, val, bri)
				wrapper.lightbulb.Brightness.Set(bri)
			}

		case wrapper.ctProperty:
			handled = append(handled, key)
			val, err := wrapper.ctConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal color_temp value %q: %s", value, err)
			} else {
				ct := int64(val)
				wrapper.log.Debugf("color_temp value %q: %f -> %d", wrapper.ctProperty, val, ct)
				wrapper.lightbulb.ColorTemp.Set(ct)
			}

		case wrapper.colorProperty:
			handled = append(handled, key)
			if !wrapper.ignoreColor {
				var vals map[string]json.RawMessage
				err := json.Unmarshal(value, &vals)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal color value %q: %s", value, err)
				} else {
					for colorKey, colorValue := range vals {
						hue, sat, x, y := wrapper.lightbulb.Color.Get()
						switch colorKey {
						case wrapper.colorXProperty:
							val, err := wrapper.colorXConverter.UnmarshalValue(colorValue)
							if err != nil {
								wrapper.log.Errorf("failed to unmarshal color.x value %q: %s", colorValue, err)
							} else {
								wrapper.log.Debugf("color.x value %q: %f", wrapper.colorXProperty, val)
								x = val
							}

						case wrapper.colorYProperty:
							val, err := wrapper.colorYConverter.UnmarshalValue(colorValue)
							if err != nil {
								wrapper.log.Errorf("failed to unmarshal color.y value %q: %s", colorValue, err)
							} else {
								wrapper.log.Debugf("color.y value %q: %f", wrapper.colorYProperty, val)
								y = val
							}

						case wrapper.colorHueProperty:
							val, err := wrapper.colorHueConverter.UnmarshalValue(colorValue)
							if err != nil {
								wrapper.log.Errorf("failed to unmarshal color.hue value %q: %s", colorValue, err)
							} else {
								wrapper.log.Debugf("color.hue value %q: %f", wrapper.colorHueProperty, val)
								hue = val
							}

						case wrapper.colorSatProperty:
							val, err := wrapper.colorSatConverter.UnmarshalValue(colorValue)
							if err != nil {
								wrapper.log.Errorf("failed to unmarshal color.sat value %q: %s", colorValue, err)
							} else {
								wrapper.log.Debugf("color.sat value %q: %f", wrapper.colorSatProperty, val)
								sat = val
							}

						default:
							wrapper.log.Errorf("unsupported color property %q: %s", colorKey, colorValue)
						}
						wrapper.lightbulb.Color.Set(hue, sat, x, y)
					}
				}
			}

		case "color_mode":
			handled = append(handled, key)
			// ignore
		}
	}
	return handled
}

func (wrapper *WrapperLight) handleAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	wrapper.log.Debugf("handling request: %s", request)
	if wrapper.requests != nil {
		reqjson := make(map[string]json.RawMessage)
		for _, val := range request.Values {
			switch val.Id {
			case wrapper.lightbulb.On.ID():
				if wrapper.onConverter != nil {
					valjson, err := wrapper.onConverter.MarshalValue(val.GetBool().GetValue())
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[wrapper.onProperty] = valjson
				} else {
					wrapper.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			case wrapper.lightbulb.Brightness.ID():
				if wrapper.briConverter != nil {
					var bri float64
					if wrapper.briConverter.ValueMax != nil {
						bri = float64(val.GetInt().GetValue()) / 100.0 * *wrapper.briConverter.ValueMax
						if bri < 0.0 {
							bri = 0.0
						} else if bri > *wrapper.briConverter.ValueMax {
							bri = *wrapper.briConverter.ValueMax
						}
					} else {
						bri = float64(val.GetInt().GetValue())
					}
					valjson, err := wrapper.briConverter.MarshalValue(bri)
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[wrapper.briProperty] = valjson
				} else {
					wrapper.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			case wrapper.lightbulb.ColorTemp.ID():
				if wrapper.ctConverter != nil {
					ct := float64(val.GetInt().GetValue())
					valjson, err := wrapper.ctConverter.MarshalValue(ct)
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[wrapper.ctProperty] = valjson
				} else {
					wrapper.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			case wrapper.lightbulb.Color.ID():
				if val.GetColor().HueSat != nil {
					if wrapper.colorHueConverter != nil && wrapper.colorSatConverter != nil {
						huejson, err := wrapper.colorHueConverter.MarshalValue(val.GetColor().GetHueSat().Hue)
						if err != nil {
							return fmt.Errorf("marshal hue %s: %s", val, err)
						}
						satjson, err := wrapper.colorSatConverter.MarshalValue(val.GetColor().GetHueSat().Sat)
						if err != nil {
							return fmt.Errorf("marshal sat %s: %s", val, err)
						}
						huesat := map[string]json.RawMessage{
							wrapper.colorHueProperty: huejson,
							wrapper.colorSatProperty: satjson,
						}
						huesatjson, err := json.Marshal(huesat)
						if err != nil {
							return fmt.Errorf("marshal feature %s: %s", val, err)
						}
						reqjson[wrapper.colorProperty] = huesatjson
					} else {
						wrapper.log.Errorf("no converter for %s", val)
						return fmt.Errorf("no converter for %s", val)
					}
				} else if val.GetColor().Xy != nil {
					if wrapper.colorXConverter != nil && wrapper.colorYConverter != nil {
						xjson, err := wrapper.colorXConverter.MarshalValue(val.GetColor().GetXy().X)
						if err != nil {
							return fmt.Errorf("marshal x %s: %s", val, err)
						}
						yjson, err := wrapper.colorYConverter.MarshalValue(val.GetColor().GetXy().Y)
						if err != nil {
							return fmt.Errorf("marshal y %s: %s", val, err)
						}
						xy := map[string]json.RawMessage{
							wrapper.colorXProperty: xjson,
							wrapper.colorYProperty: yjson,
						}
						xyjson, err := json.Marshal(xy)
						if err != nil {
							return fmt.Errorf("marshal feature %s: %s", val, err)
						}
						reqjson[wrapper.colorProperty] = xyjson
					} else {
						wrapper.log.Errorf("no converter for %s", val)
						return fmt.Errorf("no converter for %s", val)
					}
				}

			case wrapper.lightbulb.Transition.ID():
				if wrapper.transitionConverter != nil {
					transition := math.Round(float64(val.GetDuration().GetValue()) / 1000.0) // millis to seconds
					valjson, err := wrapper.transitionConverter.MarshalValue(transition)
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[wrapper.transitionProperty] = valjson
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
