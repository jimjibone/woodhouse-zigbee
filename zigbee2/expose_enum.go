package zigbee

import (
	"encoding/json"
	"fmt"
)

type EnumConverter struct {
	Values []string `json:"values"`
}

func UnmarshalEnum(data []byte) (*EnumConverter, error) {
	zc := &EnumConverter{}
	if err := json.Unmarshal(data, zc); err != nil {
		return nil, err
	}
	return zc, nil
}

func (zc *EnumConverter) String() string {
	return fmt.Sprintf("{values=%s}", zc.Values)
}

func (zc *EnumConverter) UnmarshalValue(data []byte) (string, error) {
	var value string
	err := json.Unmarshal(data, &value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (zc *EnumConverter) MarshalValue(value string) ([]byte, error) {
	return json.Marshal(value)
}
