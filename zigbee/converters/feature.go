package converters

import (
	"encoding/json"
	"fmt"
)

func NewFeature(data []byte) (map[string]Converter, error) {
	tmp := struct {
		Features []ValueInfo `json:"features"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	converters := make(map[string]Converter)
	for _, feature := range tmp.Features {
		switch feature.Type {
		case "binary":
			conv, err := NewBinary(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("binary %q: %w", feature.Property, err)
			} else {
				converters[feature.Property] = conv
			}

		case "numeric":
			conv, err := NewNumeric(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("numeric %q: %w", feature.Property, err)
			} else {
				converters[feature.Property] = conv
			}

		case "enum":
			conv, err := NewEnum(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("enum %q: %w", feature.Property, err)
			} else {
				converters[feature.Property] = conv
			}

		case "text":
			conv, err := NewText(feature.Data)
			if err != nil {
				return nil, fmt.Errorf("text %q: %w", feature.Property, err)
			} else {
				converters[feature.Property] = conv
			}

		case "composite":
			switch feature.Property {
			case "color":
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
				return nil, fmt.Errorf("unknown composite type: %s", feature)
			}

		default:
			return nil, fmt.Errorf("unknown feature type: %s", feature)
		}
	}
	return converters, nil
}
