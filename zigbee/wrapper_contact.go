package zigbee

import (
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

var _ (Wrapper) = (*WrapperContact)(nil)

type WrapperContact struct {
	log     *log.Context
	contact *services.Contact

	contactProperty  string
	contactConverter *BinaryConverter
}

func SupportsContact(info DeviceInfo) bool {
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "binary":
			switch {
			case expose.Property == "contact":
				// This is the feature we need.
				return true
			}
		}
	}
	return false
}

func NewWrapperContact(log *log.Context, dev *devices.Device) *WrapperContact {
	wrapper := &WrapperContact{
		log:     log,
		contact: services.NewContact(""),
	}
	dev.AddService(wrapper.contact)
	return wrapper
}

func (wrapper *WrapperContact) UpdateInfo(info DeviceInfo) (handled []HandledExpose) {
	var err error
	for _, expose := range info.Definition.Exposes {
		switch expose.Type {
		case "binary":
			switch {
			case expose.Property == "contact":
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.contactProperty = expose.Property
				wrapper.contactConverter, err = UnmarshalBinary(expose.Data)
				if err != nil {
					wrapper.log.Errorf("failed to unmarshal contact value: %s -- %s", err, expose)
				} else {
					wrapper.log.Debugf("contact value expose %q: %s", wrapper.contactProperty, wrapper.contactConverter)
				}
				if !wrapper.contact.Closed.IsSet() {
					wrapper.contact.Closed.Set(false)
				}
			}
		}
	}
	return handled
}

func (wrapper *WrapperContact) UpdateState(state DeviceState) (handled []string) {
	for key, value := range state.Values {
		if key == "" {
			continue
		}
		switch key {
		case wrapper.contactProperty:
			handled = append(handled, key)
			val, err := wrapper.contactConverter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal contact value %q: %s", value, err)
			} else {
				wrapper.log.Debugf("contact value %q: %t", wrapper.contactProperty, val)
				wrapper.contact.Closed.Set(val)
			}
		}
	}
	return handled
}
