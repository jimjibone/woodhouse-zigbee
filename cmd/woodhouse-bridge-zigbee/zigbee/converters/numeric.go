package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
)

var _ Converter = (*Numeric)(nil)

type Numeric struct {
	ValueMax  *float64        `json:"value_max"`  // (optional)
	ValueMin  *float64        `json:"value_min"`  // (optional)
	ValueStep *float64        `json:"value_step"` // (optional)
	Unit      *string         `json:"unit"`       // (optional)
	Presets   []NumericPreset `json:"presets"`
}

type NumericPreset struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

func NewNumeric(data []byte) (*Numeric, error) {
	zc := &Numeric{}
	if err := json.Unmarshal(data, zc); err != nil {
		return nil, err
	}
	return zc, nil
}

func (zc *Numeric) String() string {
	var msg []string
	if zc.ValueMax != nil {
		msg = append(msg, fmt.Sprintf("max=%f", *zc.ValueMax))
	}
	if zc.ValueMin != nil {
		msg = append(msg, fmt.Sprintf("min=%f", *zc.ValueMin))
	}
	if zc.ValueStep != nil {
		msg = append(msg, fmt.Sprintf("step=%f", *zc.ValueStep))
	}
	if zc.Unit != nil {
		msg = append(msg, fmt.Sprintf("unit=%s", *zc.Unit))
	}
	if len(zc.Presets) > 0 {
		msg = append(msg, fmt.Sprintf("presets=%v", zc.Presets))
	}
	return "{" + strings.Join(msg, ", ") + "}"
}

func (np *NumericPreset) String() string {
	return fmt.Sprintf("{name=%v, value=%v, desc=%v}", np.Name, np.Value, np.Description)
}

func (zc *Numeric) Unmarshal(data []byte) (*api.DeviceValue, error) {
	var value float64
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return &api.DeviceValue{
		Number: &api.NumberValue{
			Value: value,
		},
	}, nil
}

func (zc *Numeric) Marshal(value *api.DeviceValue) ([]byte, error) {
	val := &api.NumberValue{}
	if !apitools.ValueAs(value, val) {
		return nil, fmt.Errorf("invalid numeric value %v", value)
	}
	return json.Marshal(val.Value)
}
