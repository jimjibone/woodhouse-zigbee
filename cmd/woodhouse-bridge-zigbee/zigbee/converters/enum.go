package converters

import (
	"encoding/json"
	"fmt"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
)

var _ Converter = (*Enum)(nil)

type Enum struct {
	Values []string `json:"values"`
}

func NewEnum(data []byte) (*Enum, error) {
	zc := &Enum{}
	if err := json.Unmarshal(data, zc); err != nil {
		return nil, err
	}
	return zc, nil
}

func (zc *Enum) String() string {
	return fmt.Sprintf("{values=%s}", zc.Values)
}

func (zc *Enum) Unmarshal(data []byte) (*api.DeviceValue, error) {
	var value string
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return &api.DeviceValue{
		Text: &api.TextValue{
			Value: value,
		},
	}, nil
}

func (zc *Enum) Marshal(value *api.DeviceValue) ([]byte, error) {
	val := &api.TextValue{}
	if !apitools.ValueAs(value, val) {
		return nil, fmt.Errorf("invalid enum value %v", value)
	}
	return []byte(`"` + val.Value + `"`), nil
}
