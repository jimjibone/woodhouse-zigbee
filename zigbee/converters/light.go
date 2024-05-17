package converters

import (
	"encoding/json"
	"fmt"
)

func NewLight(data []byte) (map[string]Converter, error) {
	tmp := struct {
		Features []ValueInfo `json:"features"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	converters := make(map[string]Converter)
	for _, feature := range tmp.Features {
		switch feature.Name {
		case "state":
			conv, err := NewBinary(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("state %q: %w", feature.Name, err)
			} else {
				converters[feature.Property] = conv
			}

		case "brightness":
			conv, err := NewNumeric(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("brightness %q: %w", feature.Name, err)
			} else {
				converters[feature.Property] = conv
			}

		case "color_temp":
			conv, err := NewNumeric(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("color_temp %q: %w", feature.Name, err)
			} else {
				converters[feature.Property] = conv
			}

		case "color_temp_startup":
			conv, err := NewNumeric(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("color_temp_startup %q: %w", feature.Name, err)
			} else {
				converters[feature.Property] = conv
			}

		case "color_xy", "color_hs":
			if prev, found := converters[feature.Property]; !found {
				conv, err := NewColor(feature.Data)
				if err != nil {
					return nil, fmt.Errorf("color %q: %w", feature.Name, err)
				} else {
					converters[feature.Property] = conv
				}
			} else {
				if color, ok := prev.(*Color); ok {
					err := color.Update(feature.Data)
					if err != nil {
						return nil, fmt.Errorf("color %q: %w", feature.Name, err)
					}
				} else {
					return nil, fmt.Errorf("color %q: previous converter is not a color", feature.Name)
				}
			}

		default:
			return nil, fmt.Errorf("unknown feature type: %s", feature)
		}
	}
	return converters, nil
}
