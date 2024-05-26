package zigbee

import (
	"encoding/json"
	"fmt"
	"strings"
)

type NumericConverter struct {
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

func UnmarshalNumeric(data []byte) (*NumericConverter, error) {
	zc := &NumericConverter{}
	if err := json.Unmarshal(data, zc); err != nil {
		return nil, err
	}
	return zc, nil
}

func (zc *NumericConverter) String() string {
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

func (zc *NumericConverter) UnmarshalValue(data []byte) (float64, error) {
	var value float64
	err := json.Unmarshal(data, &value)
	if err != nil {
		return 0.0, err
	}
	return value, nil
}

func (zc *NumericConverter) MarshalValue(value float64) ([]byte, error) {
	return json.Marshal(value)
}
