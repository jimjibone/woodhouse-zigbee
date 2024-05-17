package converters

import (
	"encoding/json"
	"fmt"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
)

var _ Converter = (*Text)(nil)

type Text struct {
	Value string `json:"value"`
}

func NewText(data []byte) (*Text, error) {
	zc := &Text{}
	if err := json.Unmarshal(data, zc); err != nil {
		return nil, err
	}
	return zc, nil
}

func (zc *Text) String() string {
	return fmt.Sprintf("{value=%s}", zc.Value)
}

func (zc *Text) Unmarshal(data []byte) (*api.DeviceValue, error) {
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

func (zc *Text) Marshal(value *api.DeviceValue) ([]byte, error) {
	val := &api.TextValue{}
	if !apitools.ValueAs(value, val) {
		return nil, fmt.Errorf("invalid text value %v", value)
	}
	return []byte(`"` + val.Value + `"`), nil
}
