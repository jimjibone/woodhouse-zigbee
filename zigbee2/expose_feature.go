package zigbee

import (
	"encoding/json"
)

func UnmarshalFeature(data []byte) ([]ExposeInfo, error) {
	tmp := struct {
		Features []ExposeInfo `json:"features"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	return tmp.Features, nil
}
