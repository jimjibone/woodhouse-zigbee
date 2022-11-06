package shelly_v2

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Rest struct {
	IP string
}

type Shelly struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	MAC        string `json:"mac"`
	Model      string `json:"model"`
	Gen        int    `json:"gen"`
	FirmwareID string `json:"fw_id"`
	Version    string `json:"ver"`
	App        string `json:"app"`
	AuthEn     bool   `json:"auth_en"`
	AuthDomain string `json:"auth_domain"`
	Profile    string `json:"profile"`
}

func (r Rest) GetShelly() (Shelly, error) {
	req, err := http.NewRequest("GET", "http://"+r.IP+"/shelly", nil)
	if err != nil {
		return Shelly{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Shelly{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Shelly{}, err
	}
	res.Body.Close()

	var msg Shelly
	err = json.Unmarshal(body, &msg)
	if err != nil {
		return Shelly{}, err
	}
	return msg, nil
}
