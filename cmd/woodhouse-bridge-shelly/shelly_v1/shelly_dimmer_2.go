package shelly_v1

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

func init() {
	registerDevice("SHDM-2", func(hostname, ip string, client *wh.Client) Device {
		return NewShellyDimmer2(hostname, ip, client)
	})
}

// ShellyDimmer2 - device type: SHDM-2
type ShellyDimmer2 struct {
	log      *log.Context
	rest     *Rest
	hostname string
	ip       string

	client *wh.Client
	added  bool

	dev       *devices.DeviceImpl
	info      *services.Info
	online    *services.Online
	lightbulb *services.Lightbulb

	mu    sync.RWMutex
	wg    sync.WaitGroup
	close func()
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
	TransitionMs   int    `json:"transition"`      // One-shot transition, 0..5000 [ms]
}

func NewShellyDimmer2(hostname, ip string, client *wh.Client) *ShellyDimmer2 {
	ctx, close := context.WithCancel(context.Background())
	dev := &ShellyDimmer2{
		log:       log.NewContext(log.DefaultLogger, hostname, log.DebugLevel),
		rest:      NewRest(ip),
		hostname:  hostname,
		ip:        ip,
		client:    client,
		dev:       devices.NewDevice(hostname, clientsapi.Device_LIGHTBULB),
		info:      services.NewInfo(),
		online:    services.NewOnline(),
		lightbulb: services.NewLightbulb(),
		close:     close,
	}
	dev.log.Infof("created")
	dev.dev.AddService(dev.info, dev.online, dev.lightbulb)
	dev.info.Name.OnAction(dev.handleNameAction)
	dev.lightbulb.OnAction(dev.handleLightbulbAction)

	// Get the initial info and state.
	err := dev.updateInfo()
	if err != nil {
		dev.log.Warnf("failed to update info: %s", err)
	}
	err = dev.requestAndUpdateState("")
	if err != nil {
		dev.log.Warnf("failed to update state: %s", err)
	}

	dev.wg.Add(1)
	go dev.run(ctx)
	return dev
}

func (dev *ShellyDimmer2) ID() string {
	return dev.hostname
}

func (dev *ShellyDimmer2) Close() {
	dev.close()
	dev.wg.Wait()
}

func (dev *ShellyDimmer2) handleNameAction(val string) {
	dev.log.Errorf("not changing name to %s", val)
}

func (dev *ShellyDimmer2) handleLightbulbAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	var params []string

	for _, req := range request.Values {
		switch req.Id {
		case dev.lightbulb.On.ID():
			if req.GetBool() == nil {
				return services.ErrIncorrectTypeFor(dev.lightbulb.On)
			}
			if req.GetBool().GetValue() {
				params = append(params, "turn=on")
			} else {
				params = append(params, "turn=off")
			}

		case dev.lightbulb.Brightness.ID():
			if req.GetInt() == nil {
				return services.ErrIncorrectTypeFor(dev.lightbulb.Brightness)
			}
			bri := math.Max(math.Min(float64(req.GetInt().GetValue()), 100), 0)
			params = append(params, fmt.Sprintf("brightness=%.0f", bri))

		case dev.lightbulb.Transition.ID():
			if req.GetDuration() == nil {
				return services.ErrIncorrectTypeFor(dev.lightbulb.Transition)
			}
			trans := math.Max(math.Min(float64(req.GetDuration().GetValue()), 5000), 0)
			params = append(params, fmt.Sprintf("transition=%.0f", trans))
		}
	}

	if len(params) > 0 {
		reqparams := strings.Join(params, "&")
		dev.log.Infof("name: %s, sending request: %s", dev.info.Name.Get(), reqparams)
		err := dev.requestAndUpdateState(reqparams)
		if err != nil {
			dev.log.Errorf("failed to set state: %s", err)
			return err
		}
	}

	return nil
}

func (dev *ShellyDimmer2) run(ctx context.Context) {
	defer dev.wg.Done()

	infoTicker := time.NewTicker(10 * time.Second)
	defer infoTicker.Stop()

	stateTicker := time.NewTicker(time.Second)
	defer stateTicker.Stop()

	for {
		connected := false
		select {
		case <-ctx.Done():
			return

		case <-infoTicker.C:
			err := dev.updateInfo()
			if err != nil {
				dev.log.Warnf("failed to update info: %s", err)
			} else {
				connected = true
			}

		case <-stateTicker.C:
			err := dev.requestAndUpdateState("")
			if err != nil {
				dev.log.Warnf("failed to update state: %s", err)
			} else {
				connected = true
			}
		}

		// If we didn't connect then implement exponential backoff.
		dev.rest.Backoff(dev.log, ctx, connected)
	}
}

func (dev *ShellyDimmer2) updateInfo() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	settings, err := dev.rest.GetSettings()
	if err != nil {
		return err
	}

	dev.log.Debugf("settings: %s", settings.Name)

	// Add the device to the client once we've fully connected to it.
	if !dev.added {
		dev.added = true
		dev.client.AddDevice(dev.dev)
	}

	if dev.info.Name.Set(settings.Name) {
		dev.log.Infof("name: %s", settings.Name)
	}
	dev.info.Model.Set("Shelly Dimmer 2")
	dev.info.Manufacturer.Set("Shelly")
	// dev.info.SerialNumber.Set("")
	// dev.info.FirmwareVersion.Set("")
	dev.info.WebUrl.Set("http://" + dev.ip)
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	return nil
}

func (dev *ShellyDimmer2) requestAndUpdateState(params string) error {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	endpoint := "light/0"
	if params != "" {
		endpoint = endpoint + "?" + params
	}

	// Get the latest state.
	next := ShellyDimmer2State{}
	err := dev.rest.GetJSON(endpoint, &next)
	if err != nil {
		if dev.online.Online.Set(false) {
			dev.log.Infof("went offline")
		}
		return err
	}

	transition := time.Duration(next.TransitionMs) * time.Millisecond
	dev.log.Debugf("state - on: %t, bri: %d, transition: %s", next.IsOn, next.Brightness, transition)

	changed := false
	if dev.online.Online.Set(true) {
		dev.log.Infof("came online")
		changed = true
	}
	if dev.lightbulb.On.Set(next.IsOn) {
		dev.log.Infof("on: %t", next.IsOn)
		changed = true
	}
	if dev.lightbulb.Brightness.Set(int64(next.Brightness)) {
		dev.log.Infof("brightness: %d%%", next.Brightness)
		changed = true
	}
	if changed {
		// Prevent unnecessary updates when nothing changed.
		dev.online.LastSeen.Set(time.Now())
	}

	return nil
}
