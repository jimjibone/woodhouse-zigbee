package converters

import (
	"encoding/json"
	"fmt"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
)

var _ Converter = (*Binary)(nil)

type Binary struct {
	ValueOn     string  // string or bool
	ValueOff    string  // string or bool
	ValueToggle *string // (optional)
}

func NewBinary(data []byte) (*Binary, error) {
	tmp := struct {
		ValueOn     interface{} `json:"value_on"`     // string or bool
		ValueOff    interface{} `json:"value_off"`    // string or bool
		ValueToggle *string     `json:"value_toggle"` // (optional)
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	// value_on and value_off seem to be inverted for contact sensors for some
	// reason - this is the opposite to what is shown in zigbee2mqtt. So if the
	// value_on/value_off values are bools then ignore the info for this and use
	// their logical values, e.g. true == true, not true == false, etc.
	zc := &Binary{}
	if _, isBool := tmp.ValueOn.(bool); isBool {
		zc.ValueOn = "true"
	} else {
		if v, err := ParseBinaryValue(tmp.ValueOn); err != nil {
			return nil, fmt.Errorf("value_on %w", err)
		} else {
			zc.ValueOn = v
		}
	}
	if _, isBool := tmp.ValueOff.(bool); isBool {
		zc.ValueOff = "false"
	} else {
		if v, err := ParseBinaryValue(tmp.ValueOff); err != nil {
			return nil, fmt.Errorf("value_off %w", err)
		} else {
			zc.ValueOff = v
		}
	}
	zc.ValueToggle = tmp.ValueToggle
	return zc, nil
}

func (zc *Binary) String() string {
	if zc.ValueToggle != nil {
		return fmt.Sprintf("{on=%s, off=%s, toggle=%s}", zc.ValueOn, zc.ValueOff, *zc.ValueToggle)
	}
	return fmt.Sprintf("{on=%s, off=%s}", zc.ValueOn, zc.ValueOff)
}

func (zc *Binary) Unmarshal(data []byte) (*api.DeviceValue, error) {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	v, err := ParseBinaryValue(value)
	if err != nil {
		return nil, err
	}
	val := &api.DeviceValue{
		Bool: &api.BoolValue{
			Value: false,
		},
	}
	switch v {
	case zc.ValueOn:
		val.Bool.Value = true
	case zc.ValueOff:
		val.Bool.Value = false
	default:
		return nil, fmt.Errorf("invalid binary value %v", v)
	}
	return val, nil
}

func (zc *Binary) Marshal(value *api.DeviceValue) ([]byte, error) {
	val := &api.BoolValue{}
	if !apitools.ValueAs(value, val) {
		return nil, fmt.Errorf("invalid binary value %v", value)
	}
	if val.Value {
		// return []byte(`"` + zc.ValueOn + `"`), nil
		return json.Marshal(zc.ValueOn)
	}
	// return []byte(`"` + zc.ValueOff + `"`), nil
	return json.Marshal(zc.ValueOff)
}

func ParseBinaryValue(value interface{}) (string, error) {
	switch vv := value.(type) {
	case bool:
		if vv {
			return "true", nil
		}
		return "false", nil
	case string:
		return vv, nil
	}
	return "", fmt.Errorf("invalid binary value %v", value)
}
