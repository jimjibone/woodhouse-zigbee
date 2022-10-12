package zigbee

import (
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/jimjibone/woodhouse-4/api/go"
)

type Exposed struct {
	Access         Access       `json:"access"` // (optional)
	Description    string       `json:"description"`
	Name           string       `json:"name"`
	PrefixProperty string       `json:"-"` // Defined if this was part of a composite value
	Property       string       `json:"property"`
	Type           string       `json:"type"`
	Value          ExposedValue `json:"value"`
}

func (e *Exposed) String() string {
	name := e.Name
	if name == "" {
		name = "no name"
	}
	description := e.Description
	if description == "" {
		description = "no description"
	}
	property := e.Property
	if property == "" {
		property = "no property"
	}
	typeStr := e.Type
	if typeStr == "" {
		typeStr = "no type"
	}
	return fmt.Sprintf("%s: %s (%s, %s)", name, description, property, typeStr)
}

func (e *Exposed) LongString(indent string) string {
	name := e.Name
	if name == "" {
		name = "no name"
	}
	description := e.Description
	if description == "" {
		description = "no description"
	}
	property := e.Property
	if property == "" {
		property = "no property"
	}
	typeStr := e.Type
	if typeStr == "" {
		typeStr = "no type"
	}
	return fmt.Sprintf("%s%s: %s (%s, %s):\n%s", indent, name, description, property, typeStr, e.Value.LongString(indent+"  "))
}

func (e *Exposed) UnmarshalJSON(data []byte) error {
	// Determine the value type.
	tmpType := struct {
		Type string
	}{}
	if err := json.Unmarshal(data, &tmpType); err != nil {
		return err
	}

	// Get the exposer type.
	var exposedValue ExposedValue
	switch tmpType.Type {
	case "binary":
		exposedValue = &ExposedBinary{}
	case "numeric":
		exposedValue = &ExposedNumeric{}
	case "enum":
		exposedValue = &ExposedEnum{}
	case "text":
		exposedValue = &ExposedText{}
	case "composite":
		exposedValue = &ExposedComposite{}
	case "light", "switch", "fan", "cover", "lock", "climate":
		exposedValue = &ExposedFeatures{}
	}

	// Unmarshal the message into the exposer first.
	if err := json.Unmarshal(data, exposedValue); err != nil {
		return err
	}

	// Unmarshal the message into this message.
	type tmpExpose Exposed
	if err := json.Unmarshal(data, (*tmpExpose)(e)); err != nil {
		return err
	}

	// Set the exposer.
	e.Value = exposedValue

	return nil
}

type ExposedValue interface {
	exposed()
	fmt.Stringer
	LongString(indent string) string
	GetValue(state interface{}) *api.DeviceValue
	GetJSON(*api.DeviceValue) interface{}
}

var (
	_ (ExposedValue) = (*ExposedBinary)(nil)
	_ (ExposedValue) = (*ExposedNumeric)(nil)
	_ (ExposedValue) = (*ExposedEnum)(nil)
	_ (ExposedValue) = (*ExposedText)(nil)
	_ (ExposedValue) = (*ExposedComposite)(nil)
	_ (ExposedValue) = (*ExposedFeatures)(nil)
)

type ExposedBinary struct {
	ValueOn     string  // string or bool
	ValueOff    string  // string or bool
	ValueToggle *string // (optional)
}

func (e *ExposedBinary) UnmarshalJSON(data []byte) error {
	tmp := struct {
		ValueOn     interface{} `json:"value_on"`     // string or bool
		ValueOff    interface{} `json:"value_off"`    // string or bool
		ValueToggle *string     `json:"value_toggle"` // (optional)
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if v, err := parseBinaryValue(tmp.ValueOn); err != nil {
		return fmt.Errorf("value_on %w", err)
	} else {
		e.ValueOn = v
	}
	if v, err := parseBinaryValue(tmp.ValueOff); err != nil {
		return fmt.Errorf("value_off %w", err)
	} else {
		e.ValueOff = v
	}
	e.ValueToggle = tmp.ValueToggle
	return nil
}

func (e *ExposedBinary) exposed() {}

func (e *ExposedBinary) String() string {
	return e.LongString("")
}

func (e *ExposedBinary) LongString(indent string) string {
	if e.ValueToggle != nil {
		return fmt.Sprintf("%son=%v, off=%v, toggle=%v", indent, e.ValueOn, e.ValueOff, *e.ValueToggle)
	}
	return fmt.Sprintf("%son=%v, off=%v", indent, e.ValueOn, e.ValueOff)
}

func parseBinaryValue(v interface{}) (string, error) {
	switch vv := v.(type) {
	case bool:
		if vv {
			return "true", nil
		}
		return "false", nil
	case string:
		return vv, nil
	}
	return "", fmt.Errorf("invalid binary value %v", v)
}

func (e *ExposedBinary) GetValue(state interface{}) *api.DeviceValue {
	val := &api.DeviceValue{
		Bool: &api.BoolValue{
			Value: false,
		},
	}
	switch vv := state.(type) {
	case bool:
		val.Bool.Value = vv
	case string:
		val.Bool.Value = e.ValueOn == vv
	default:
		fmt.Printf("ERROR: unexpected state type %+v for binary: %s", state, e)
	}
	return val
}

func (e *ExposedBinary) GetJSON(req *api.DeviceValue) interface{} {
	if req.Bool != nil {
		if req.Bool.Value {
			return e.ValueOn
		}
		return e.ValueOff
	}
	return nil
}

type ExposedNumeric struct {
	ValueMax  *float64               `json:"value_max"`  // (optional)
	ValueMin  *float64               `json:"value_min"`  // (optional)
	ValueStep *float64               `json:"value_step"` // (optional)
	Unit      *string                `json:"unit"`       // (optional)
	Presets   []ExposedNumericPreset `json:"presets"`
}

type ExposedNumericPreset struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

func (e *ExposedNumeric) exposed() {}

func (e *ExposedNumeric) String() string {
	return e.LongString("")
}

func (e *ExposedNumeric) LongString(indent string) string {
	var msg []string
	if e.ValueMax != nil {
		msg = append(msg, fmt.Sprintf("max=%f", *e.ValueMax))
	}
	if e.ValueMin != nil {
		msg = append(msg, fmt.Sprintf("min=%f", *e.ValueMin))
	}
	if e.ValueStep != nil {
		msg = append(msg, fmt.Sprintf("step=%f", *e.ValueStep))
	}
	if e.Unit != nil {
		msg = append(msg, fmt.Sprintf("unit=%s", *e.Unit))
	}
	if len(e.Presets) > 0 {
		msg = append(msg, fmt.Sprintf("presets=%v", e.Presets))
	}
	return indent + strings.Join(msg, ", ")
}

func (e *ExposedNumericPreset) String() string {
	return fmt.Sprintf("name=%v, value=%v, desc=%v", e.Name, e.Value, e.Description)
}

func (e *ExposedNumeric) GetValue(state interface{}) *api.DeviceValue {
	val := &api.DeviceValue{
		Number: &api.NumberValue{
			Value: 0,
		},
	}
	switch vv := state.(type) {
	case float64:
		val.Number.Value = vv
	default:
		fmt.Printf("ERROR: unexpected state type %+v for numeric: %s", state, e)
	}
	return val
}

func (e *ExposedNumeric) GetJSON(req *api.DeviceValue) interface{} {
	if req.Number != nil {
		return req.Number.Value
	}
	return nil
}

type ExposedEnum struct {
	Values []string `json:"values"`
}

func (e *ExposedEnum) exposed() {}

func (e *ExposedEnum) String() string {
	return e.LongString("")
}

func (e *ExposedEnum) LongString(indent string) string {
	return fmt.Sprintf("%svalues=%s", indent, e.Values)
}

func (e *ExposedEnum) GetValue(state interface{}) *api.DeviceValue {
	// TODO: add this
	val := &api.DeviceValue{
		Number: &api.NumberValue{
			// Value: false,
		},
	}
	return val
}

func (e *ExposedEnum) GetJSON(req *api.DeviceValue) interface{} {
	// TODO: add this
	if req.Number != nil {
		return req.Number.Value
	}
	return nil
}

type ExposedText struct {
	Value string `json:"value"`
}

func (e *ExposedText) exposed() {}

func (e *ExposedText) String() string {
	return e.LongString("")
}

func (e *ExposedText) LongString(indent string) string {
	return fmt.Sprintf("%svalue=%s", indent, e.Value)
}

func (e *ExposedText) GetValue(state interface{}) *api.DeviceValue {
	// TODO: add this
	val := &api.DeviceValue{
		Number: &api.NumberValue{
			// Value: false,
		},
	}
	return val
}

func (e *ExposedText) GetJSON(req *api.DeviceValue) interface{} {
	// TODO: add this
	if req.Number != nil {
		return req.Number.Value
	}
	return nil
}

type ExposedComposite struct {
	Features []*Exposed `json:"features"`
}

func (e *ExposedComposite) exposed() {}

func (e *ExposedComposite) String() string {
	return e.LongString("")
}

func (e *ExposedComposite) LongString(indent string) string {
	msg := fmt.Sprintf("%scomposite=%d:", indent, len(e.Features))
	for _, feat := range e.Features {
		msg += "\n" + feat.LongString(indent+"  ")
	}
	return msg
}

func (e *ExposedComposite) GetValue(state interface{}) *api.DeviceValue {
	return nil
}

func (e *ExposedComposite) GetJSON(req *api.DeviceValue) interface{} {
	return nil
}

type ExposedFeatures struct {
	Features []*Exposed `json:"features"`
}

func (e *ExposedFeatures) exposed() {}

func (e *ExposedFeatures) String() string {
	return e.LongString("")
}

func (e *ExposedFeatures) LongString(indent string) string {
	msg := fmt.Sprintf("%sfeatures=%d:", indent, len(e.Features))
	for _, feat := range e.Features {
		msg += "\n" + feat.LongString(indent+"  ")
	}
	return msg
}

func (e *ExposedFeatures) GetValue(state interface{}) *api.DeviceValue {
	return nil
}

func (e *ExposedFeatures) GetJSON(req *api.DeviceValue) interface{} {
	return nil
}
