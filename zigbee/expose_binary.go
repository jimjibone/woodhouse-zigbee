package zigbee

import (
	"encoding/json"
	"fmt"
)

type BinaryConverter struct {
	ValueOn     string  // string or bool
	ValueOff    string  // string or bool
	ValueToggle *string // (optional)
}

func UnmarshalBinary(data []byte) (*BinaryConverter, error) {
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
	zc := &BinaryConverter{}
	if _, isBool := tmp.ValueOn.(bool); isBool {
		zc.ValueOn = "true"
	} else {
		if v, err := ParseBinaryConverterValue(tmp.ValueOn); err != nil {
			return nil, fmt.Errorf("value_on %w", err)
		} else {
			zc.ValueOn = v
		}
	}
	if _, isBool := tmp.ValueOff.(bool); isBool {
		zc.ValueOff = "false"
	} else {
		if v, err := ParseBinaryConverterValue(tmp.ValueOff); err != nil {
			return nil, fmt.Errorf("value_off %w", err)
		} else {
			zc.ValueOff = v
		}
	}
	zc.ValueToggle = tmp.ValueToggle
	return zc, nil
}

func (zc *BinaryConverter) String() string {
	if zc.ValueToggle != nil {
		return fmt.Sprintf("{on=%s, off=%s, toggle=%s}", zc.ValueOn, zc.ValueOff, *zc.ValueToggle)
	}
	return fmt.Sprintf("{on=%s, off=%s}", zc.ValueOn, zc.ValueOff)
}

func (zc *BinaryConverter) UnmarshalValue(data []byte) (bool, error) {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return false, err
	}
	v, err := ParseBinaryConverterValue(value)
	if err != nil {
		return false, err
	}
	switch v {
	case zc.ValueOn:
		return true, nil
	case zc.ValueOff:
		return false, nil
	}
	return false, fmt.Errorf("invalid binaryConverter value %v", v)
}

func (zc *BinaryConverter) MarshalValue(value bool) ([]byte, error) {
	if value {
		return json.Marshal(zc.ValueOn)
	}
	return json.Marshal(zc.ValueOff)
}

func ParseBinaryConverterValue(value interface{}) (string, error) {
	switch vv := value.(type) {
	case bool:
		if vv {
			return "true", nil
		}
		return "false", nil
	case string:
		return vv, nil
	}
	return "", fmt.Errorf("invalid binaryConverter value %v", value)
}
