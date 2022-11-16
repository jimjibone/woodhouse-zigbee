package zigbee_old

import (
	"fmt"

	api "github.com/jimjibone/woodhouse-4/api/go"
)

type ZigbeeValue interface {
	Value() *api.DeviceValue
}

var (
	_ ZigbeeValue = (*ZigbeeBool)(nil)
	_ ZigbeeValue = (*ZigbeeNumber)(nil)
	_ ZigbeeValue = (*ZigbeeText)(nil)
	_ ZigbeeValue = (*ZigbeeColor)(nil)
)

type ZigbeeBool bool

func (z ZigbeeBool) Value() *api.DeviceValue {
	return &api.DeviceValue{
		Bool: &api.BoolValue{
			Value: bool(z),
		},
	}
}

type ZigbeeNumber float64

func (z ZigbeeNumber) Value() *api.DeviceValue {
	return &api.DeviceValue{
		Number: &api.NumberValue{
			Value: float64(z),
		},
	}
}

type ZigbeeText string

func (z ZigbeeText) Value() *api.DeviceValue {
	return &api.DeviceValue{
		Text: &api.TextValue{
			Value: string(z),
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
