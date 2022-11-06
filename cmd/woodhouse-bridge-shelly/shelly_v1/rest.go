package shelly_v1

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Rest struct {
	IP string
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

func (r Rest) GetStatus() (DeviceStatus, error) {
	req, err := http.NewRequest("GET", "http://"+r.IP+"/status", nil)
	if err != nil {
		return DeviceStatus{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceStatus{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return DeviceStatus{}, err
	}
	res.Body.Close()

	var devicestatus DeviceStatus
	err = json.Unmarshal(body, &devicestatus)
	if err != nil {
		return DeviceStatus{}, err
	}
	return devicestatus, nil
}

func (r Rest) GetSettings() (DeviceSettings, error) {
	req, err := http.NewRequest("GET", "http://"+r.IP+"/settings", nil)
	if err != nil {
		return DeviceSettings{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceSettings{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return DeviceSettings{}, err
	}
	res.Body.Close()

	var devicesettings DeviceSettings
	err = json.Unmarshal(body, &devicesettings)
	if err != nil {
		return DeviceSettings{}, err
	}
	return devicesettings, nil
}

func (r Rest) GetJSON(endpoint string, v interface{}) error {
	url := "http://" + r.IP + "/" + endpoint
	// log.Printf("GET %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()

	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
}
