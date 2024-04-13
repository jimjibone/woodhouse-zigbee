package shelly_v1

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jimjibone/woodhouse-4/log"
)

type Rest struct {
	ip          string
	minBackoff  time.Duration
	maxBackoff  time.Duration
	lastBackoff time.Duration
}

func NewRest(ip string) *Rest {
	return &Rest{
		ip:          ip,
		minBackoff:  time.Second,
		maxBackoff:  32 * time.Second,
		lastBackoff: 0,
	}
}

func (r *Rest) GetShelly() (Shelly, error) {
	req, err := http.NewRequest("GET", "http://"+r.ip+"/shelly", nil)
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

func (r *Rest) GetStatus() (DeviceStatus, error) {
	req, err := http.NewRequest("GET", "http://"+r.ip+"/status", nil)
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

func (r *Rest) GetSettings() (DeviceSettings, error) {
	req, err := http.NewRequest("GET", "http://"+r.ip+"/settings", nil)
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

func (r *Rest) GetJSON(endpoint string, v interface{}) error {
	url := "http://" + r.ip + "/" + endpoint
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

// Implements an exponential backoff by sleeping the goroutine for an increasing
// amount of time, up to the maxBackoff, unless reset is true when it will
// return the backoff to minBackoff.
func (rest *Rest) Backoff(log *log.Context, ctx context.Context, reset bool) {
	if reset {
		rest.lastBackoff = rest.minBackoff
	} else {
		rest.lastBackoff = rest.lastBackoff * 2
	}
	if rest.lastBackoff <= 0 {
		rest.lastBackoff = rest.minBackoff
	}
	if rest.lastBackoff > rest.maxBackoff {
		rest.lastBackoff = rest.maxBackoff
	}
	log.Debugf("backoff for %s", rest.lastBackoff)
	timer := time.NewTimer(rest.lastBackoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
