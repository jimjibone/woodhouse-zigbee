package zigbee

import (
	"slices"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/attributes"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

type WrapperGeneric struct {
	log     *log.Context
	generic *services.Generic

	numerics map[string]*genericNumeric
}

type genericNumeric struct {
	converter *NumericConverter
	attribute *attributes.Float
}

func NewWrapperGeneric(log *log.Context, dev *devices.Device) *WrapperGeneric {
	wrapper := &WrapperGeneric{
		log:     log,
		generic: services.NewGeneric("generic"),

		numerics: make(map[string]*genericNumeric),
	}
	dev.AddService(wrapper.generic)
	return wrapper
}

func (wrapper *WrapperGeneric) UpdateInfo(info DeviceInfo, ignore []HandledExpose) (handled []HandledExpose) {
	// var err error
	for _, expose := range info.Definition.Exposes {
		// Skip exposes that have already been handled.
		if slices.Contains(ignore, HandledExpose{expose.Type, expose.Property}) {
			continue
		}

		switch expose.Type {
		case "numeric":
			converter, err := UnmarshalNumeric(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic numeric value: %s -- %s", err, expose)
			} else {
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.log.Debugf("generic numeric expose %q: %s", expose.Property, converter)

				attribute := attributes.NewFloat(expose.Property, clientsapi.Permissions_PERM_READONLY, attributes.Optional, 0, 0, 0, clientsapi.Unit_UNIT_UNDEFINED)
				wrapper.generic.AddAttribute(attribute)

				if converter.ValueMin != nil && converter.ValueMax != nil {
					if converter.ValueStep != nil {
						attribute.SetLimits(*converter.ValueMin, *converter.ValueMax, *converter.ValueStep)
					} else {
						attribute.SetLimits(*converter.ValueMin, *converter.ValueMax, 0)
					}
				}
				if !attribute.IsSet() {
					attribute.Set(0.0)
				}

				wrapper.numerics[expose.Property] = &genericNumeric{
					converter: converter,
					attribute: attribute,
				}
			}

			// 	case "binary":
			// 		switch {
			// 		case expose.Property == "contact":
			// 			handled = append(handled, HandledExpose{expose.Type, expose.Property})
			// 			wrapper.contactProperty = expose.Property
			// 			wrapper.contactConverter, err = UnmarshalBinary(expose.Data)
			// 			if err != nil {
			// 				wrapper.log.Errorf("failed to unmarshal contact value: %s -- %s", err, expose)
			// 			} else {
			// 				wrapper.log.Debugf("contact value expose %q: %s", wrapper.contactProperty, wrapper.contactConverter)
			// 			}
			// 			wrapper.contact.Closed.Set(false)
			// 		}
		}
	}
	return handled
}

func (wrapper *WrapperGeneric) UpdateState(state DeviceState, ignore []string) (handled []string) {
	for key, value := range state.Values {
		if numeric, found := wrapper.numerics[key]; found {
			handled = append(handled, key)
			val, err := numeric.converter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic %q value %q: %s", key, value, err)
			} else {
				wrapper.log.Debugf("generic value %q: %f", key, val)
				numeric.attribute.Set(val)
			}
		}

		// 	switch key {
		// 	case wrapper.contactProperty:
		// 		handled = append(handled, key)
		// 		val, err := wrapper.contactConverter.UnmarshalValue(value)
		// 		if err != nil {
		// 			wrapper.log.Errorf("failed to unmarshal contact value %q: %s", value, err)
		// 		} else {
		// 			wrapper.log.Debugf("contact value %q: %t", wrapper.contactProperty, val)
		// 			wrapper.contact.Closed.Set(val)
		// 		}
		// 	}
	}
	return handled
}
