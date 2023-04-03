package shelly_v1

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/wh"
)

func init() {
	registerDevice("SHDM-2", func(hostname, ip string) Device {
		return &ShellyDimmer2{
			rest:     Rest{IP: ip},
			hostname: hostname,
			ip:       ip,
		}
	})
}

// ShellyDimmer2 - device type: SHDM-2
type ShellyDimmer2 struct {
	comms    *wh.BridgeComms
	rest     Rest
	hostname string
	ip       string
	name     string
	online   bool
	lastSeen time.Time
	state    ShellyDimmer2State
}

type ShellyDimmer2State struct {
	IsOn           bool   `json:"ison"`            // Whether the channel is turned ON or OFF
	Source         string `json:"source"`          // Source of the last command
	HasTimer       bool   `json:"has_timer"`       // Whether a timer is currently armed for this channel
	TimerStarted   int    `json:"timer_started"`   // Unix timestamp of timer start; 0 if timer inactive or time not synced
	TimerDuration  int    `json:"timer_duration"`  // Timer duration, s
	TimerRemaining int    `json:"timer_remaining"` // experimental If there is an active timer, shows seconds until timer elapses; 0 otherwise
	Mode           string `json:"mode"`            // Always white
	Brightness     int    `json:"brightness"`      // Output brightness, 1..100
	Transition     int    `json:"transition"`      // One-shot transition, 0..5000 [ms]
}

func (d *ShellyDimmer2) Init(comms *wh.BridgeComms) {
	d.comms = comms
}

func (d *ShellyDimmer2) SendFullUpdate() {
	d.UpdateInfo()
	d.UpdateState(true)
}

func (d *ShellyDimmer2) UpdateInfo() {
	// status, err := d.rest.GetStatus()
	// if err != nil {
	// 	log.Printf("ERROR: device %s failed to get status: %s", d.hostname, err)
	// 	return
	// }

	settings, err := d.rest.GetSettings()
	if err != nil {
		log.Printf("ERROR: device %s failed to get settings: %s", d.hostname, err)
		return
	}
	d.name = settings.Name
	log.Printf("received %s settings: %s", d.hostname, settings.Name)

	err = d.comms.SendInfo(&api.DeviceInfo{
		DeviceId:    d.hostname,
		Name:        d.name,
		Description: "Shelly Dimmer 2",
		Url:         "http://" + d.ip,
	})
	if err != nil {
		log.Printf("ERROR: device %s: failed to send info: %s", d.hostname, err)
	}
}

func (d *ShellyDimmer2) UpdateState(fullUpdate bool) {
	err := d.sendAndUpdateState("", fullUpdate)
	if err != nil {
		log.Printf("ERROR: device %s failed to get state: %s", d.hostname, err)
	}
}

func (d *ShellyDimmer2) sendAndUpdateState(params string, fullUpdate bool) error {
	endpoint := "light/0"
	if params != "" {
		endpoint = endpoint + "?" + params
	}

	// Get the latest state.
	next := ShellyDimmer2State{}
	err := d.rest.GetJSON(endpoint, &next)
	if err != nil {
		if d.online {
			d.online = false
			err = d.comms.SendState(&api.DeviceState{
				DeviceId:   d.hostname,
				FullUpdate: fullUpdate,
				Online:     d.online,
				LastSeen:   apitools.TimeToTimestamp(d.lastSeen),
			})
			if err != nil {
				log.Printf("ERROR: device %s: failed to send state: %s", d.hostname, err)
			}
		}
		return err
	}

	// Check for differences.
	d.online = true
	d.lastSeen = time.Now()
	update := &api.DeviceState{
		DeviceId:   d.hostname,
		FullUpdate: fullUpdate,
		Online:     d.online,
		LastSeen:   apitools.TimeToTimestamp(d.lastSeen),
	}
	if fullUpdate || next.IsOn != d.state.IsOn {
		update.Values = append(update.Values, &api.DeviceValue{
			Name: "On",
			Bool: &api.BoolValue{
				Value: next.IsOn,
			},
		})
	}
	if fullUpdate || next.Brightness != d.state.Brightness {
		update.Values = append(update.Values, &api.DeviceValue{
			Name: "Brightness",
			Number: &api.NumberValue{
				Value: float64(next.Brightness),
			},
		})
	}
	if fullUpdate || next.Transition != d.state.Transition {
		update.Values = append(update.Values, &api.DeviceValue{
			Name: "Transition",
			Number: &api.NumberValue{
				Value: float64(next.Transition),
			},
		})
	}

	// Update the stored state.
	d.state = next

	if len(update.Values) > 0 {
		log.Printf("device %s: name: %s, on: %t, bri: %d, trans: %d", d.hostname, d.name, d.state.IsOn, d.state.Brightness, d.state.Transition)
		err = d.comms.SendState(update)
		if err != nil {
			log.Printf("ERROR: device %s: failed to send state: %s", d.hostname, err)
		}
	}

	return nil
}

func (d *ShellyDimmer2) HandleRequest(request *api.DeviceRequest) error {
	var params []string

	for _, req := range request.Values {
		switch req.Name {
		case "On":
			if req.Bool == nil {
				return fmt.Errorf("On value must be a bool")
			}
			if req.Bool.Value {
				params = append(params, "turn=on")
			} else {
				params = append(params, "turn=off")
			}

		case "Brightness":
			if req.Number == nil {
				return fmt.Errorf("Brightness value must be a number")
			}
			bri := math.Max(math.Min(float64(req.Number.Value), 100), 0)
			params = append(params, fmt.Sprintf("brightness=%.0f", bri))

		case "Transition":
			if req.Number == nil {
				return fmt.Errorf("Transition value must be a number")
			}
			trans := math.Max(math.Min(float64(req.Number.Value), 5000), 0)
			params = append(params, fmt.Sprintf("transition=%.0f", trans))

		default:
			return fmt.Errorf("value %s not recognised", req.Name)
		}
	}

	if len(params) > 0 {
		reqparams := strings.Join(params, "&")
		log.Printf("device %s: name: %s, sending request: %s", d.hostname, d.name, reqparams)
		err := d.sendAndUpdateState(reqparams, false)
		if err != nil {
			log.Printf("ERROR: device %s failed to set state: %s", d.hostname, err)
			return err
		}
	}

	return nil
}
