package zigbee

import (
	"encoding/json"
	"fmt"
	"time"
)

type DeviceState struct {
	LastSeen time.Time
	Values   map[string]json.RawMessage
}

func (d DeviceState) String() string {
	return fmt.Sprintf("last_seen:%s, values:%s", d.LastSeen, d.Values)
}

func (d *DeviceState) UnmarshalJSON(data []byte) error {
	// Unmarshal the last_seen timestamp.
	tmp := struct {
		LastSeen time.Time `json:"last_seen"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	d.LastSeen = tmp.LastSeen

	// Unmarshal the values.
	values := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	delete(values, "last_seen")
	d.Values = values

	return nil
}
