package converters

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ValueInfo struct {
	Access      ValueInfoAccess `json:"access"` // (optional)
	Description string          `json:"description"`
	Name        string          `json:"name"`
	Property    string          `json:"property"`
	Type        string          `json:"type"`
	Data        []byte          `json:"-"`
}

func (e ValueInfo) String() string {
	return fmt.Sprintf("access:%s, desc:%s, name:%s, prop:%s, type:%s, data:%s",
		e.Access,
		e.Description,
		e.Name,
		e.Property,
		e.Type,
		e.Data,
	)
}

func (e *ValueInfo) UnmarshalJSON(data []byte) error {
	type Tmp ValueInfo
	var tmp Tmp
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*e = ValueInfo(tmp)

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	delete(raw, "access")
	delete(raw, "description")
	delete(raw, "name")
	delete(raw, "property")
	delete(raw, "type")
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	e.Data = data

	return nil
}

type ValueInfoAccess struct {
	Pub bool // The property can be found in the published state of this device.
	Set bool // The property can be set with a `/set` command.
	Get bool // The property can be retrieved with a /get command (when this is is true, Pub will also be true).
}

func (a ValueInfoAccess) String() string {
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

func (a *ValueInfoAccess) UnmarshalJSON(data []byte) error {
	var tmp uint
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	a.Pub = tmp&0b001 > 0
	a.Set = tmp&0b010 > 0
	a.Get = tmp&0b100 > 0
	return nil
}
