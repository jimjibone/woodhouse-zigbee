package zigbee

import (
	"encoding/json"
	"strings"
)

type Access struct {
	Pub bool // The property can be found in the published state of this device.
	Set bool // The property can be set with a `/set` command.
	Get bool // The property can be retrieved with a /get command (when this is is true, Pub will also be true).
}

func (a Access) String() string {
	var parts []string
	if a.Pub {
		parts = append(parts, "pub")
	}
	if a.Set {
		parts = append(parts, "set")
	}
	if a.Get {
		parts = append(parts, "get")
	}
	return strings.Join(parts, "+")
}

func (a *Access) UnmarshalJSON(data []byte) error {
	var tmp uint
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	a.Pub = tmp&0b001 > 0
	a.Set = tmp&0b010 > 0
	a.Get = tmp&0b100 > 0
	return nil
}
