package zigbee

import (
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperMotion)(nil)

type WrapperMotion struct {
	log *log.Context
	srv *services.Motion

	occupancyProperty  string
	occupancyConverter *BinaryConverter
}

func SupportsMotion(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "binary":
			switch {
			case expose.Property == "occupancy":
				// This is the feature we need.
				return true
			}
		}
	}
	return false
}

func NewWrapperMotion(log *log.Context, dev *devices.Device) *WrapperMotion {
	wrapper := &WrapperMotion{
		log: log,
		srv: services.NewMotion(""),
	}
	dev.AddService(wrapper.srv)
	return wrapper
}

func (wrapper *WrapperMotion) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	var err error
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "binary":
			switch {
			case expose.Property == "occupancy":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.occupancyProperty = expose.Property
				wrapper.occupancyConverter, err = UnmarshalBinary(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal occupancy value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("occupancy value expose %q: %s", wrapper.occupancyProperty, wrapper.occupancyConverter)
				}
				if !wrapper.srv.Motion.IsSet() {
					wrapper.srv.Motion.Set(false)
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperMotion) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		if key == "" {
			continue
		}
		switch key {
		case wrapper.occupancyProperty:
			handled = append(handled, key)
			val, err := wrapper.occupancyConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal occupancy value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("occupancy value %q: %t", wrapper.occupancyProperty, val)
				wrapper.srv.Motion.Set(val)
			}
		}
	}
	return handled
}
