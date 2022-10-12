package zigbee

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type DeviceState struct {
	LastSeen time.Time
	Values   map[string]interface{}
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
	// Unmarshal the last seen timestamp.
	lastseen := struct {
		LastSeen time.Time `json:"last_seen"`
	}{}
	if err := json.Unmarshal(data, &lastseen); err != nil {
		return err
	}

	// Unmarshal the values.
	values := make(map[string]interface{})
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	delete(values, "last_seen")

	// Update.
	d.LastSeen = lastseen.LastSeen
	d.Values = values

	return nil
}
