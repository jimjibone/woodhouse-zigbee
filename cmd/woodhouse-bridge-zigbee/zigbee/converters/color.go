package converters

import (
	"encoding/json"
	"fmt"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
)

var _ Converter = (*Color)(nil)

// {
// 	"description": "Color of this light expressed as hue/saturation",
// 	"features": [
// 		{
// 			"access": 7,
// 			"name": "hue",
// 			"property": "hue",
// 			"type": "numeric"
// 		},
// 		{
// 			"access": 7,
// 			"name": "saturation",
// 			"property": "saturation",
// 			"type": "numeric"
// 		}
// 	],
// 	"name": "color_hs",
// 	"property": "color",
// 	"type": "composite"
// }

type Color struct {
	HasHueSat bool
	HasXY     bool
}

func NewColor(data []byte) (*Color, error) {
	zc := &Color{}
	err := zc.Update(data)
	if err != nil {
		return nil, err
	}
	return zc, nil
}

func (zc *Color) Update(data []byte) error {
	tmp := struct {
		Features []struct {
			Name     string `json:"name"`
			Property string `json:"property"`
			Type     string `json:"type"`
		} `json:"features"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	for _, feature := range tmp.Features {
		switch feature.Name {
		case "hue", "saturation":
			zc.HasHueSat = true
		case "x", "y":
			zc.HasXY = true
		}
	}
	return nil
}

func (zc *Color) String() string {
	return fmt.Sprintf("{hs=%t, xy=%t}", zc.HasHueSat, zc.HasXY)
}

func (zc *Color) Unmarshal(data []byte) (*api.DeviceValue, error) {
	value := struct {
		Hue float32 `json:"hue"`
		Sat float32 `json:"saturation"`
		X   float32 `json:"x"`
		Y   float32 `json:"y"`
	}{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return &api.DeviceValue{
		Color: &api.ColorValue{
			Hue: value.Hue,
			Sat: value.Sat,
		},
	}, nil
}

func (zc *Color) Marshal(value *api.DeviceValue) ([]byte, error) {
	val := &api.ColorValue{}
	if !apitools.ValueAs(value, val) {
		return nil, fmt.Errorf("invalid color value %v", value)
	}
	return json.Marshal(struct {
		Hue float32 `json:"hue"`
		Sat float32 `json:"saturation"`
		// X   float32 `json:"x"`
		// Y   float32 `json:"y"`
	}{
		Hue: val.Hue,
		Sat: val.Sat,
	})
}
