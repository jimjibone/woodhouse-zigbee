package zigbee

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/wh/v1/devices"
	"github.com/jimjibone/woodhouse-core/wh/v1/devices/attributes"
	"github.com/jimjibone/woodhouse-core/wh/v1/devices/services"
)

type WrapperGeneric struct {
	log      *log.Context
	generic  *services.Generic
	requests func(payload []byte)

	binaries map[string]*genericBinary
	enums    map[string]*genericEnum
	numerics map[string]*genericNumeric
}

type genericBinary struct {
	converter *BinaryConverter
	attribute *attributes.Bool
}

type genericEnum struct {
	converter *EnumConverter
	attribute *attributes.Enum
}

type genericNumeric struct {
	converter *NumericConverter
	attribute *attributes.Float
}

func NewWrapperGeneric(log *log.Context, dev *devices.Device, requests func(payload []byte)) *WrapperGeneric {
	wrapper := &WrapperGeneric{
		log:      log,
		generic:  services.NewGeneric("generic"),
		requests: requests,

		binaries: make(map[string]*genericBinary),
		enums:    make(map[string]*genericEnum),
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
		case "binary":
			converter, err := UnmarshalBinary(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic binary value: %s -- %s", err, expose)
			} else {
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.log.Debugf("generic binary expose %q: %s", expose.Property, converter)

				perms := clientsapi.Permissions_PERM_READONLY
				if expose.Access.Set {
					perms = clientsapi.Permissions_PERM_READWRITE
				}

				attribute := attributes.NewBool(expose.Property, perms, attributes.Optional)

				if attribute.Perms() == clientsapi.Permissions_PERM_READWRITE {
					attribute.OnAction(func(val bool) {
						wrapper.log.Debugf("handling binary request: %t", val)

						valjson, err := converter.MarshalValue(val)
						if err != nil {
							panic(fmt.Sprintf("failed to marshal binary valjson: %s --- %s", valjson, err))
						}
						reqjson := map[string]json.RawMessage{
							expose.Property: valjson,
						}

						data, err := json.Marshal(reqjson)
						if err != nil {
							panic(fmt.Sprintf("failed to marshal binary reqjson: %s --- %s", reqjson, err))
						}
						wrapper.requests(data)
					})
				}

				wrapper.generic.AddAttribute(attribute)

				if !attribute.IsSet() {
					attribute.Set(false)
				}

				wrapper.binaries[expose.Property] = &genericBinary{
					converter: converter,
					attribute: attribute,
				}
			}

		case "enum":
			converter, err := UnmarshalEnum(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic enum value: %s -- %s", err, expose)
			} else {
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.log.Debugf("generic enum expose %q: %s", expose.Property, converter)

				perms := clientsapi.Permissions_PERM_READONLY
				if expose.Access.Set {
					perms = clientsapi.Permissions_PERM_READWRITE
				}

				attribute := attributes.NewEnum(expose.Property, perms, attributes.Optional)

				if attribute.Perms() == clientsapi.Permissions_PERM_READWRITE {
					attribute.OnAction(func(val string) {
						wrapper.log.Debugf("handling enum request: %q", val)

						valjson, err := converter.MarshalValue(val)
						if err != nil {
							panic(fmt.Sprintf("failed to marshal enum valjson: %s --- %s", valjson, err))
						}
						reqjson := map[string]json.RawMessage{
							expose.Property: valjson,
						}

						data, err := json.Marshal(reqjson)
						if err != nil {
							panic(fmt.Sprintf("failed to marshal enum reqjson: %s --- %s", reqjson, err))
						}
						wrapper.requests(data)
					})
				}

				wrapper.generic.AddAttribute(attribute)

				attribute.SetOptions(converter.Values)
				if !attribute.IsSet() {
					attribute.Set("")
				}

				wrapper.enums[expose.Property] = &genericEnum{
					converter: converter,
					attribute: attribute,
				}
			}

		case "numeric":
			converter, err := UnmarshalNumeric(expose.Data)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic numeric value: %s -- %s", err, expose)
			} else {
				handled = append(handled, HandledExpose{expose.Type, expose.Property})
				wrapper.log.Debugf("generic numeric expose %q: %s", expose.Property, converter)

				perms := clientsapi.Permissions_PERM_READONLY
				if expose.Access.Set {
					perms = clientsapi.Permissions_PERM_READWRITE
				}

				attribute := attributes.NewFloat(expose.Property, perms, attributes.Optional, 0, 0, 0, clientsapi.Unit_UNIT_UNDEFINED)

				if attribute.Perms() == clientsapi.Permissions_PERM_READWRITE {
					attribute.OnAction(func(val float64) {
						wrapper.log.Debugf("handling numeric request: %f", val)

						valjson, err := converter.MarshalValue(val)
						if err != nil {
							panic(fmt.Sprintf("failed to marshal numeric valjson: %s --- %s", valjson, err))
						}
						reqjson := map[string]json.RawMessage{
							expose.Property: valjson,
						}

						data, err := json.Marshal(reqjson)
						if err != nil {
							panic(fmt.Sprintf("failed to marshal numeric reqjson: %s --- %s", reqjson, err))
						}
						wrapper.requests(data)
					})
				}

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
		}
	}
	return handled
}

func (wrapper *WrapperGeneric) UpdateState(state DeviceState, ignore []string) (handled []string) {
	for key, value := range state.Values {
		if key == "" {
			continue
		}
		if binary, found := wrapper.binaries[key]; found {
			handled = append(handled, key)
			val, err := binary.converter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic %q value %q: %s", key, value, err)
			} else {
				wrapper.log.Debugf("generic value %q: %f", key, val)
				binary.attribute.Set(val)
			}
		} else if enum, found := wrapper.enums[key]; found {
			handled = append(handled, key)
			val, err := enum.converter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic %q value %q: %s", key, value, err)
			} else {
				wrapper.log.Debugf("generic value %q: %f", key, val)
				enum.attribute.Set(val)
			}
		} else if numeric, found := wrapper.numerics[key]; found {
			handled = append(handled, key)
			val, err := numeric.converter.UnmarshalValue(value)
			if err != nil {
				wrapper.log.Errorf("failed to unmarshal generic %q value %q: %s", key, value, err)
			} else {
				wrapper.log.Debugf("generic value %q: %f", key, val)
				numeric.attribute.Set(val)
			}
		}
	}
	return handled
}
