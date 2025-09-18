package zigbee

import (
	"encoding/json"
	"fmt"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperCover)(nil)

type WrapperCover struct {
	log      *log.Context
	requests func(payload []byte)

	cover *services.Cover

	positionProperty  string
	positionConverter *NumericConverter

	stateProperty  string
	stateConverter *EnumConverter
}

func SupportsCover(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "cover":
			// This is the feature we need.
			return true
		}
	}
	return false
}

func NewWrapperCover(log *log.Context, dev *devices.Device, requests func(payload []byte)) *WrapperCover {
	wrapper := &WrapperCover{
		log:      log,
		cover:    services.NewCover(""),
		requests: requests,
	}
	wrapper.cover.OnAction(wrapper.handleAction)
	dev.AddService(wrapper.cover)
	return wrapper
}

func (wrapper *WrapperCover) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "cover":
			handled = append(handled, HandledExpose{expose.Type, expose.Property})
			feature, err := UnmarshalFeature(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal cover: %s -- %s", err, expose)
			} else {
				for _, featureExpose := range feature {
					switch featureExpose.Name {
					case "position":
						wrapper.positionProperty = featureExpose.Property
						wrapper.positionConverter, err = UnmarshalNumeric(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal cover position: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("cover position expose %q: %s", wrapper.positionProperty, wrapper.positionConverter)
						}
						if !wrapper.cover.Position.IsSet() {
							wrapper.cover.Position.Set(0)
						}

					case "state":
						wrapper.stateProperty = featureExpose.Property
						wrapper.stateConverter, err = UnmarshalEnum(featureExpose.Data)
						if err != nil {
							wrapper.log.Errorf("failed to unmarshal action value: %s -- %s", err, expose)
						} else {
							wrapper.log.Debugf("action value expose %q: %s", wrapper.stateProperty, wrapper.stateConverter)
						}
						if !wrapper.cover.State.IsSet() {
							wrapper.cover.State.Set("")
						}
						wrapper.cover.State.SetOptions(wrapper.stateConverter.Values)

					default:
						wrapper.log.Warnf("unsupported cover expose %q: %s", featureExpose.Name, featureExpose)
					}
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperCover) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		switch key {
		case wrapper.positionProperty:
			handled = append(handled, key)
			val, err := wrapper.positionConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal position value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("position value %q: %v", wrapper.positionProperty, val)
				wrapper.cover.Position.Set(int64(val))
			}

		case wrapper.stateProperty:
			handled = append(handled, key)
			val, err := wrapper.stateConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal state value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("state value %q: %v", wrapper.stateProperty, val)
				wrapper.cover.State.Set(val)
			}
		}
	}
	return handled
}

func (wrapper *WrapperCover) handleAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	wrapper.log.Debugf("handling request: %s", request)
	if wrapper.requests != nil {
		reqjson := make(map[string]json.RawMessage)
		for _, val := range request.Values {
			switch val.Id {
			case wrapper.cover.Position.ID():
				if wrapper.positionConverter != nil {
					valjson, err := wrapper.positionConverter.MarshalValue(float64(val.GetInt().GetValue()))
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[wrapper.positionProperty] = valjson
				} else {
					wrapper.log.Errorf("no converter for %s", val)
					return fmt.Errorf("no converter for %s", val)
				}

			case wrapper.cover.State.ID():
				if wrapper.stateConverter != nil {
					valjson, err := wrapper.stateConverter.MarshalValue(val.GetText().GetValue())
					if err != nil {
						return fmt.Errorf("marshal %s: %s", val, err)
					}
					reqjson[wrapper.stateProperty] = valjson
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
