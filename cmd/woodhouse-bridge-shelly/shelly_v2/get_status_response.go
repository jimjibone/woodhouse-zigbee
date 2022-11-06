package shelly_v2

import (
	"encoding/json"
	"strings"
)

type GetStatusResponse struct {
	BLE   struct{} `json:"ble"`
	Cloud struct {
		Connected bool `json:"connected"`
	} `json:"cloud"`
	Inputs []GetStatusResponseInput `json:"-"`
	MQTT   struct {
		Connected bool `json:"connected"`
	} `json:"mqtt"`
	Scripts  []GetStatusResponseScript `json:"-"`
	Switches []GetStatusResponseSwitch `json:"-"`
	System   struct {
		MAC              string `json:"mac"`
		RestartRequired  bool   `json:"restart_required"`
		Time             string `json:"time"`
		UnixTime         int    `json:"unixtime"`
		Uptime           int    `json:"uptime"`
		RamSize          int    `json:"ram_size"`
		RamFree          int    `json:"ram_free"`
		FsSize           int    `json:"fs_size"`
		FsFree           int    `json:"fs_free"`
		ConfigRevision   int    `json:"cfg_rev"`
		AvailableUpdates map[string]struct {
			Version string `json:"version"`
		} `json:"available_updates"`
	} `json:"sys"`
	WiFi json.RawMessage `json:"wifi"`
	WS   json.RawMessage `json:"ws"`
}

type GetStatusResponseInput struct {
	ID    int  `json:"id"`
	State bool `json:"state"`
}

type GetStatusResponseScript struct {
	ID      int  `json:"id"`
	Running bool `json:"running"`
}

type GetStatusResponseSwitch struct {
	ID            int      `json:"id"`
	Source        *string  `json:"source"`
	Output        *bool    `json:"output"`
	AveragePower  *float64 `json:"apower"`
	Voltage       *float64 `json:"voltage"`
	Current       *float64 `json:"current"`
	AverageEnergy *struct {
		Total           float64   `json:"total"`
		ByMinute        []float64 `json:"by_minute"`
		MinuteTimestamp int       `json:"minute_ts"`
	} `json:"aenergy"`
	Temperature *struct {
		Centigrade float64 `json:"tC"`
		Farenheit  float64 `json:"tF"`
	} `json:"temperature"`
}

func (m *GetStatusResponse) UnmarshalJSON(data []byte) error {
	type Tmp GetStatusResponse
	tmp := Tmp{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	rawFields := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &rawFields)
	if err != nil {
		return err
	}

	for field, data := range rawFields {
		if strings.Contains(field, "input") {
			msg := GetStatusResponseInput{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				return err
			}
			tmp.Inputs = append(tmp.Inputs, msg)
		} else if strings.Contains(field, "script") {
			msg := GetStatusResponseScript{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				return err
			}
			tmp.Scripts = append(tmp.Scripts, msg)
		} else if strings.Contains(field, "switch") {
			msg := GetStatusResponseSwitch{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				return err
			}
			tmp.Switches = append(tmp.Switches, msg)
		}
	}

	*m = GetStatusResponse(tmp)

	return nil
}
