package zigbee_old

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type DeviceState struct {
	LastSeen time.Time
	Values   map[string]ZigbeeValue
}

func (d DeviceState) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("last_seen:%s", d.LastSeen))
	for n, v := range d.Values {
		parts = append(parts, fmt.Sprintf("%s:%v", n, v))
	}
	return strings.Join(parts, ", ")
}

func (d DeviceState) LongString(indent string) string {
	return indent + d.String()
}

func (d *DeviceState) UnmarshalJSON(data []byte) error {
	// Unmarshal the values.
	values := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}

	d.Values = make(map[string]ZigbeeValue)

	// Unmarshal values. Some values have dedicated destination types.
	for name, value := range values {
		switch name {
		case "last_seen":
			// Unmarshal the last seen timestamp.
			if err := json.Unmarshal(value, &d.LastSeen); err != nil {
				return err
			}

		case "color":
			// Unmarshal the color struct.
			var color ZigbeeColor
			if err := json.Unmarshal(value, &color); err != nil {
				return err
			}
			d.Values["color"] = color

		case "update":
			// Ignore.

		default:
			// Any other value type.
			var val interface{}
			if err := json.Unmarshal(value, &val); err != nil {
				return err
			}
			switch v := val.(type) {
			case bool:
				d.Values[name] = ZigbeeBool(v)

			case float64:
				d.Values[name] = ZigbeeNumber(v)

			case string:
				d.Values[name] = ZigbeeText(v)

			case nil:
				log.Printf("WARN: value %s is nil: %s", name, value)

			default:
				return fmt.Errorf("unable to unmarshal %T value %s: %+v", val, name, val)
			}
		}
	}

	return nil
}
