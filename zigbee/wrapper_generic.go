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

	binaries map[string]*genericBinary
	numerics map[string]*genericNumeric
}

type genericBinary struct {
	converter *BinaryConverter
	attribute *attributes.Bool
}

type genericNumeric struct {
	converter *NumericConverter
	attribute *attributes.Float
}

func NewWrapperGeneric(log *log.Context, dev *devices.Device) *WrapperGeneric {
	wrapper := &WrapperGeneric{
		log:     log,
		generic: services.NewGeneric("generic"),

		binaries: make(map[string]*genericBinary),
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

		case "binary":
			converter, err := UnmarshalBinary(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic binary value: %s -- %s", err, expose)
			} else {
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.log.Debugf("generic binary expose %q: %s", expose.Property, converter)

				attribute := attributes.NewBool(expose.Property, clientsapi.Permissions_PERM_READONLY, attributes.Optional)
				wrapper.generic.AddAttribute(attribute)

				if !attribute.IsSet() {
					attribute.Set(false)
				}

				wrapper.binaries[expose.Property] = &genericBinary{
					converter: converter,
					attribute: attribute,
				}
			}
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
		} else if binary, found := wrapper.binaries[key]; found {
			handled = append(handled, key)
			val, err := binary.converter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic %q value %q: %s", key, value, err)
			} else {
				wrapper.log.Debugf("generic value %q: %f", key, val)
				binary.attribute.Set(val)
			}
		}
	}
	return handled
}
