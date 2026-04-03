package zigbee

import (
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperPresence)(nil)

type WrapperPresence struct {
	log *log.Context
	srv *services.Presence

	motionProperty  string
	motionConverter *EnumConverter

	presenceProperty  string
	presenceConverter *BinaryConverter

	distanceProperty  string
	distanceConverter *NumericConverter
}

func SupportsPresence(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "binary":
			switch {
			case expose.Property == "presence":
				// This is the feature we need.
				return true
			}
		}
	}
	return false
}

func NewWrapperPresence(log *log.Context, dev *devices.Device) *WrapperPresence {
	wrapper := &WrapperPresence{
		log: log,
		srv: services.NewPresence(""),
	}
	dev.AddService(wrapper.srv)
	return wrapper
}

func (wrapper *WrapperPresence) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	var err error
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "enum":
			switch {
			case expose.Property == "movement":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.motionProperty = expose.Property
				wrapper.motionConverter, err = UnmarshalEnum(expose.Data)
				if err != nil {
					wrapper.log.Errorf("presence: failed to unmarshal motion value: %s -- %s", err, expose)
				} else {
					wrapper.log.Infof("presence: motion value expose %q: %s", wrapper.motionProperty, wrapper.motionConverter)
				}
				if !wrapper.srv.Motion.IsSet() {
					wrapper.srv.Motion.Set(false)
				}
			}

		case "binary":
			switch {
			case expose.Property == "presence":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.presenceProperty = expose.Property
				wrapper.presenceConverter, err = UnmarshalBinary(expose.Data)
				if err != nil {
					wrapper.log.Errorf("presence: failed to unmarshal presence value: %s -- %s", err, expose)
				} else {
					wrapper.log.Infof("presence: presence value expose %q: %s", wrapper.presenceProperty, wrapper.presenceConverter)
				}
				if !wrapper.srv.Presence.IsSet() {
					wrapper.srv.Presence.Set(false)
				}
			}

		case "numeric":
			switch {
			case expose.Property == "target_distance":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.distanceProperty = expose.Property
				wrapper.distanceConverter, err = UnmarshalNumeric(expose.Data)
				if err != nil {
					wrapper.log.Errorf("presence: failed to unmarshal distance value: %s -- %s", err, expose)
				} else {
					wrapper.log.Infof("presence: distance value expose %q: %s", wrapper.distanceProperty, wrapper.distanceConverter)
				}
				if !wrapper.srv.Distance.IsSet() {
					wrapper.srv.Distance.Set(0.0)
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperPresence) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		if key == "" {
			continue
		}
		switch key {
		case wrapper.motionProperty:
			handled = append(handled, key)
			val, err := wrapper.motionConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal motion value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("presence: motion value %q: %q", wrapper.motionProperty, val)
				if val == "movement" {
					wrapper.srv.Motion.Set(true)
				} else {
					wrapper.srv.Motion.Set(false)
				}
			}

		case wrapper.presenceProperty:
			handled = append(handled, key)
			val, err := wrapper.presenceConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal presence value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("presence: presence value %q: %t", wrapper.presenceProperty, val)
				wrapper.srv.Presence.Set(val)
			}

		case wrapper.distanceProperty:
			handled = append(handled, key)
			val, err := wrapper.distanceConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal distance value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("presence: distance value %q: %f", wrapper.distanceProperty, val)
				wrapper.srv.Distance.Set(val)
			}
		}
	}
	return handled
}
