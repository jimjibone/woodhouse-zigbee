package converters

import (
	"fmt"

	api "github.com/jimjibone/woodhouse-4/api/go"
)

type ZigbeeBool bool

func ConvertBool(v bool) *api.DeviceValue {
	return &api.DeviceValue{
		Bool: &api.BoolValue{
			Value: bool(v),
		},
	}
}

func ConvertNumber(v float64) *api.DeviceValue {
	return &api.DeviceValue{
		Number: &api.NumberValue{
			Value: float64(v),
		},
	}
}

func ConvertText(t string) *api.DeviceValue {
	return &api.DeviceValue{
		Text: &api.TextValue{
			Value: string(t),
		},
	}
}

type ZigbeeColor struct {
	Hue float32 `json:"hue"`
	Sat float32 `json:"saturation"`
}

func (d ZigbeeColor) String() string {
	return fmt.Sprintf("{hue:%.1f, sat:%.1f}", d.Hue, d.Sat)
}

func (z ZigbeeColor) Value() *api.DeviceValue {
	return &api.DeviceValue{
		Color: &api.ColorValue{
			Hue: z.Hue,
			Sat: z.Sat,
		},
	}
}
