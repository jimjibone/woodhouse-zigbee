package shelly_v2

import (
	"encoding/json"
	"strings"
)

type GetConfigResponse struct {
	BLE struct {
		Enable bool `json:"enable"`
	} `json:"ble"`
	Cloud struct {
		Enable bool   `json:"enable"`
		Server string `json:"server"`
	} `json:"cloud"`
	Eth struct {
		Enable     bool   `json:"enable"`
		IPv4Mode   string `json:"ipv4mode"`
		IP         string `json:"ip"`
		Netmask    string `json:"netmask"`
		Gateway    string `json:"gw"`
		Nameserver string `json:"nameserver"`
	} `json:"eth"`
	Inputs []GetConfigResponseInput `json:"-"`
	MQTT   struct {
		Enable      bool   `json:"enable"`
		Server      string `json:"server"`
		ClientID    string `json:"client_id"`
		User        string `json:"user"`
		Pass        string `json:"pass"`
		TopicPrefix string `json:"topic_prefix"`
		RpcNtf      bool   `json:"rpc_ntf"`
		StatusNpf   bool   `json:"status_npf"`
	} `json:"mqtt"`
	Scripts  []GetConfigResponseScript `json:"-"`
	Switches []GetConfigResponseSwitch `json:"-"`
	System   struct {
		Device struct {
			Name         string `json:"name"`
			MAC          string `json:"mac"`
			FirmwareID   string `json:"fw_id"`
			Discoverable bool   `json:"discoverable"`
			EcoMode      bool   `json:"eco_mode"`
			Profile      string `json:"profile"`
		} `json:"device"`
		Location struct {
			Timezone string  `json:"tz"`
			Lat      float64 `json:"lat"`
			Lon      float64 `json:"lon"`
		} `json:"location"`
		Debug struct {
			MQTT struct {
				Enable bool `json:"enable"`
			} `json:"mqtt"`
			Websocket struct {
				Enable bool `json:"enable"`
			} `json:"websocket"`
			UDP struct {
				Addr string `json:"addr"`
			} `json:"udp"`
		} `json:"debug"`
		UIData json.RawMessage `json:"ui_data"`
		RPCUDP struct {
			DestAddr   string `json:"dst_addr"`
			ListenPort string `json:"listen_port"`
		} `json:"rpc_udp"`
		SNTP struct {
			Server string `json:"server"`
		} `json:"sntp"`
		ConfigRevision int `json:"cfg_rev"`
	} `json:"sys"`
	WiFi json.RawMessage `json:"wifi"`
	WS   json.RawMessage `json:"ws"`
}

type GetConfigResponseInput struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Invert bool   `json:"invert"`
}

type GetConfigResponseScript struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
}

type GetConfigResponseSwitch struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	InMode       string  `json:"in_mode"`
	InitialState string  `json:"initial_state"`
	AutoOn       bool    `json:"auto_on"`
	AutoOnDelay  float64 `json:"auto_on_delay"` // seconds
	AutoOff      bool    `json:"auto_off"`
	AutoOffDelay float64 `json:"auto_off_delay"` // seconds
	PowerLimit   float64 `json:"power_limit"`    // watts
	VolatgeLimit float64 `json:"volatge_limit"`  // volts
	CurrentLimit float64 `json:"current_limit"`  // amps
}

func (m *GetConfigResponse) UnmarshalJSON(data []byte) error {
	type Tmp GetConfigResponse
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
			msg := GetConfigResponseInput{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				return err
			}
			tmp.Inputs = append(tmp.Inputs, msg)
		} else if strings.Contains(field, "script") {
			msg := GetConfigResponseScript{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				return err
			}
			tmp.Scripts = append(tmp.Scripts, msg)
		} else if strings.Contains(field, "switch") {
			msg := GetConfigResponseSwitch{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				return err
			}
			tmp.Switches = append(tmp.Switches, msg)
		}
	}

	*m = GetConfigResponse(tmp)

	return nil
}
