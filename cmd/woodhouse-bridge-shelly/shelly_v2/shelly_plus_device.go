package shelly_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/wh"
)

const (
	minBackoff = time.Second
	maxBackoff = 30 * time.Second
)

// ShellyPlusDevice - device type: Plus2PM
type ShellyPlusDevice struct {
	comms           *wh.BridgeComms
	cancel          func()
	wg              sync.WaitGroup
	hostname        string
	ip              string
	name            string
	description     string
	connMu          sync.RWMutex
	conn            *websocket.Conn
	inputs          map[int]*InputValue
	scripts         map[int]*ScriptValue
	switches        map[int]*SwitchValue
	lastBackoff     time.Time
	lastRestart     time.Time
	backoffDuration time.Duration
}

func NewShellyPlusDevice(hostname, ip, name, app string) Device {
	description := "Shelly Plus Device"
	switch app {
	case "Plus1PM":
		description = "Shelly Plus 1PM"
	case "Plus2PM":
		description = "Shelly Plus 2PM"
	default:
		log.Printf("WARN: unknown app %q for %s", app, hostname)
	}
	return &ShellyPlusDevice{
		hostname:    hostname,
		ip:          ip,
		name:        name,
		description: description,
		inputs:      make(map[int]*InputValue),
		scripts:     make(map[int]*ScriptValue),
		switches:    make(map[int]*SwitchValue),
	}
}

type InputValue struct {
	ID        int
	Name      string
	Type      string
	Invert    bool
	Timestamp time.Time
	State     bool
}

func (v *InputValue) GetName() string {
	if v.Name == "" {
		return fmt.Sprintf("Input %d", v.ID)
	}
	return v.Name
}

type ScriptValue struct {
	ID        int
	Name      string
	Timestamp time.Time
	Enable    bool
	Running   bool
}

func (v *ScriptValue) GetName() string {
	if v.Name == "" {
		return fmt.Sprintf("Script %d", v.ID)
	}
	return v.Name
}

type SwitchValue struct {
	ID            int
	Name          string
	Timestamp     time.Time
	State         bool
	AveragePower  float64
	Voltage       float64
	Current       float64
	AverageEnergy struct {
		Total           float64
		ByMinute        []float64
		MinuteTimestamp int
	}
	Temperature float64 // Centigrade
}

func (v *SwitchValue) GetName() string {
	if v.Name == "" {
		return fmt.Sprintf("Switch %d", v.ID)
	}
	return v.Name
}

func (d *ShellyPlusDevice) Init(comms *wh.BridgeComms) {
	d.comms = comms
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.wg.Add(1)
	go d.run(ctx)
}

func (d *ShellyPlusDevice) Close() {
	d.cancel()
	d.wg.Wait()
}

func (d *ShellyPlusDevice) run(ctx context.Context) {
	defer d.wg.Done()

	log.Printf("%s started", d.hostname)
	defer log.Printf("%s finished", d.hostname)

	for {
		// Try to connect.
		log.Printf("%s creating websocket", d.hostname)
		conn := d.connect()
		if conn != nil {
			d.connMu.Lock()
			d.conn = conn
			d.connMu.Unlock()

			// Close the websocket when the context is cancelled.
			go func() {
				<-ctx.Done()
				log.Printf("%s closing websocket", d.hostname)
				d.connMu.Lock()
				if d.conn != nil {
					d.conn.Close()
					d.conn = nil
				}
				d.connMu.Unlock()
			}()

			// Receive updates and send requests.
			d.recv(ctx, conn)
		}

		// Check if we're done.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Backoff for a short while before the next connection attempt.
		d.backoff(ctx)
	}
}

func (d *ShellyPlusDevice) connect() *websocket.Conn {
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/rpc", d.ip), nil)
	if err != nil {
		log.Println("ERROR: dial:", err)
		conn.Close()
		return nil
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"id":100, "src":"woodhouse", "method":"Shelly.GetConfig"}`))
	if err != nil {
		log.Println("ERROR: write:", err)
		conn.Close()
		return nil
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("ERROR: read:", err)
		conn.Close()
		return nil
	}
	// log.Printf("%s --> recv: %s", d.hostname, message)

	var frame ResponseFrame
	err = json.Unmarshal(message, &frame)
	if err != nil {
		log.Println("ERROR: unmarshal:", err)
		conn.Close()
		return nil
	}
	// log.Printf("config frame: %+v", frame)

	if frame.Result != nil {
		var configResponse GetConfigResponse
		err = json.Unmarshal(frame.Result, &configResponse)
		if err != nil {
			log.Println("ERROR: unmarshal:", err)
			conn.Close()
			return nil
		}
		// log.Printf("system config: %+v", configResponse)

		d.name = configResponse.System.Device.Name

		for _, val := range configResponse.Inputs {
			d.inputs[val.ID] = &InputValue{
				ID:     val.ID,
				Name:   val.Name,
				Type:   val.Type,
				Invert: val.Invert,
			}
		}
		for _, val := range configResponse.Scripts {
			d.scripts[val.ID] = &ScriptValue{
				ID:     val.ID,
				Name:   val.Name,
				Enable: val.Enable,
			}
		}
		for _, val := range configResponse.Switches {
			d.switches[val.ID] = &SwitchValue{
				ID:   val.ID,
				Name: val.Name,
			}
		}
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"id":101, "src":"woodhouse", "method":"Shelly.GetStatus"}`))
	if err != nil {
		log.Println("ERROR: write:", err)
		conn.Close()
		return nil
	}

	_, message, err = conn.ReadMessage()
	if err != nil {
		log.Println("ERROR: read:", err)
		conn.Close()
		return nil
	}
	// log.Printf("%s --> recv: %s", d.hostname, message)

	err = json.Unmarshal(message, &frame)
	if err != nil {
		log.Println("ERROR: unmarshal:", err)
		conn.Close()
		return nil
	}
	// log.Printf("status frame: %+v", frame)

	if frame.Result != nil {
		var statusResponse GetStatusResponse
		err = json.Unmarshal(frame.Result, &statusResponse)
		if err != nil {
			log.Println("ERROR: unmarshal:", err)
			conn.Close()
			return nil
		}
		// log.Printf("system status: %+v", statusResponse)

		for _, val := range statusResponse.Inputs {
			if prev, found := d.inputs[val.ID]; found {
				prev.Timestamp = time.Now()
				prev.State = val.State
			}
		}
		for _, val := range statusResponse.Scripts {
			if prev, found := d.scripts[val.ID]; found {
				prev.Timestamp = time.Now()
				prev.Running = val.Running
			}
		}
		for _, val := range statusResponse.Switches {
			if prev, found := d.switches[val.ID]; found {
				prev.Timestamp = time.Now()
				if val.Output != nil {
					prev.State = *val.Output
				}
				if val.AveragePower != nil {
					prev.AveragePower = *val.AveragePower
				}
				if val.Voltage != nil {
					prev.Voltage = *val.Voltage
				}
				if val.Current != nil {
					prev.Current = *val.Current
				}
				if val.AverageEnergy != nil {
					prev.AverageEnergy.Total = val.AverageEnergy.Total
					prev.AverageEnergy.ByMinute = val.AverageEnergy.ByMinute
					prev.AverageEnergy.MinuteTimestamp = val.AverageEnergy.MinuteTimestamp
				}
				if val.Temperature != nil {
					prev.Temperature = val.Temperature.Centigrade
				}
			}
		}
	}

	var ids []int
	for _, val := range d.inputs {
		ids = append(ids, val.ID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		val := d.inputs[id]
		log.Printf("new input %d, name: %q, type: %s, invert: %t, ts: %s, state: %t", id, val.Name, val.Type, val.Invert, val.Timestamp.Format("2006/01/02 15:04:05"), val.State)
	}

	ids = nil
	for _, val := range d.scripts {
		ids = append(ids, val.ID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		val := d.scripts[id]
		log.Printf("new script %d, name: %q, ts: %s, enable: %t, running %t", id, val.Name, val.Timestamp.Format("2006/01/02 15:04:05"), val.Enable, val.Running)
	}

	ids = nil
	for _, val := range d.switches {
		ids = append(ids, val.ID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		val := d.switches[id]
		log.Printf("new switch %d, name: %q, ts: %s, state: %t, temp: %0.1f °C, voltage: %.1f V, current: %.1f A, avg.power: %.1f W, avg.energy.total: %.3f Wh", id, val.Name, val.Timestamp.Format("2006/01/02 15:04:05"), val.State, val.Temperature, val.Voltage, val.Current, val.AveragePower, val.AverageEnergy.Total)
	}

	d.UpdateState(nil)

	return conn
}

func (d *ShellyPlusDevice) recv(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ERROR: %s read: %s", d.hostname, err)
			return
		}
		// log.Printf("%s --> recv: %s", d.hostname, message)

		switch DetectFrameType(message) {
		case UnknownFrameType:
			log.Printf("WARN: %s unknown frame type for message: %s", d.hostname, message)

		case ResponseFrameType:
			log.Printf("%s --> recv response frame: %s", d.hostname, message)
			// var frame ResponseFrame
			// err = json.Unmarshal(message, &frame)
			// if err != nil {
			// 	log.Printf("WARN: %s unmarshal: %s, from message: %s", d.hostname, err, message)
			// } else {
			// 	log.Printf("%s --> recv response frame: %+v", d.hostname, frame)
			// }

		case NotificationFrameType:
			var frame NotificationFrame
			err = json.Unmarshal(message, &frame)
			if err != nil {
				log.Printf("WARN: %s unmarshal: %s, from message: %s", d.hostname, err, message)
			} else {
				// log.Printf("%s --> recv notification frame: %+v", d.hostname, frame)
				if frame.NotifyStatus != nil {
					d.UpdateState(frame.NotifyStatus)
				}
			}
		}
	}
}

func (sd *ShellyPlusDevice) backoff(ctx context.Context) {
	// Reset the backoff duration if the backoff has not been used for a
	// suitable amount of time.
	dt := time.Since(sd.lastRestart)
	if dt > sd.backoffDuration {
		log.Printf("backoff reset after %s", dt)
		sd.backoffDuration = minBackoff
	}
	sd.lastBackoff = time.Now()
	log.Printf("starting backoff for %s", sd.backoffDuration)
	timer := time.NewTimer(sd.backoffDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	log.Printf("backoff finished")
	sd.backoffDuration = sd.backoffDuration * 2
	if sd.backoffDuration > maxBackoff {
		sd.backoffDuration = maxBackoff
	}
	sd.lastRestart = time.Now()
}

func (d *ShellyPlusDevice) SendFullUpdate() {
	d.UpdateInfo()
	d.UpdateState(nil)
}

func (d *ShellyPlusDevice) UpdateInfo() {
	err := d.comms.SendInfo(&api.DeviceInfo{
		DeviceId:    d.hostname,
		Name:        d.name,
		Description: d.description,
		Url:         "http://" + d.ip,
	})
	if err != nil {
		log.Printf("ERROR: device %s: failed to send info: %s", d.hostname, err)
	}
}

func (d *ShellyPlusDevice) UpdateState(next *NotifyStatus) {
	update := &api.DeviceState{
		DeviceId:   d.hostname,
		FullUpdate: next == nil,
		Values:     []*api.DeviceValue{},
	}
	if next == nil {
		// Send current inputs.
		for _, val := range d.inputs {
			update.Values = append(update.Values, &api.DeviceValue{
				Name: val.GetName(),
				Bool: &api.BoolValue{
					Value: val.State,
				},
			})
		}

		// Send current scripts.
		for _, val := range d.scripts {
			update.Values = append(update.Values, &api.DeviceValue{
				Name: val.GetName(),
				Bool: &api.BoolValue{
					Value: val.Running,
				},
			})
		}

		// Send current switches.
		for _, val := range d.switches {
			update.Values = append(update.Values, &api.DeviceValue{
				Name: val.GetName(),
				Bool: &api.BoolValue{
					Value: val.State,
				},
			})
		}
	} else {
		// Update matching inputs.
		for _, val := range next.Inputs {
			if prev, found := d.inputs[val.ID]; found {
				prev.Timestamp = time.Now()
				prev.State = val.State
				update.Values = append(update.Values, &api.DeviceValue{
					Name: prev.GetName(),
					Bool: &api.BoolValue{
						Value: val.State,
					},
				})
				log.Printf("device %s: input %d, name: %q, type: %s, invert: %t, ts: %s, state: %t", d.hostname, prev.ID, prev.Name, prev.Type, prev.Invert, prev.Timestamp.Format("2006/01/02 15:04:05"), prev.State)
			}
		}

		// Update matching scripts.
		for _, val := range next.Scripts {
			if prev, found := d.scripts[val.ID]; found {
				prev.Timestamp = time.Now()
				prev.Running = val.Running
				update.Values = append(update.Values, &api.DeviceValue{
					Name: prev.GetName(),
					Bool: &api.BoolValue{
						Value: val.Running,
					},
				})
				log.Printf("device %s: script %d, name: %q, enable: %t, running: %t", d.hostname, prev.ID, prev.Name, prev.Enable, prev.Running)
			}
		}

		// Update matching switches.
		for _, val := range next.Switches {
			if prev, found := d.switches[val.ID]; found {
				prev.Timestamp = time.Now()
				if val.Output != nil {
					prev.State = *val.Output
				}
				if val.AveragePower != nil {
					prev.AveragePower = *val.AveragePower
				}
				if val.Voltage != nil {
					prev.Voltage = *val.Voltage
				}
				if val.Current != nil {
					prev.Current = *val.Current
				}
				if val.AverageEnergy != nil {
					prev.AverageEnergy.Total = val.AverageEnergy.Total
					prev.AverageEnergy.ByMinute = val.AverageEnergy.ByMinute
					prev.AverageEnergy.MinuteTimestamp = val.AverageEnergy.MinuteTimestamp
				}
				if val.Temperature != nil {
					prev.Temperature = val.Temperature.Centigrade
				}
				if val.Output != nil {
					update.Values = append(update.Values, &api.DeviceValue{
						Name: prev.GetName(),
						Bool: &api.BoolValue{
							Value: *val.Output,
						},
					})
				}
				log.Printf("device %s: switch %d, name: %q, ts: %s, state: %t, temp: %0.1f °C, voltage: %.1f V, current: %.1f A, avg.power: %.1f W, avg.energy.total: %.3f Wh", d.hostname, prev.ID, prev.Name, prev.Timestamp.Format("2006/01/02 15:04:05"), prev.State, prev.Temperature, prev.Voltage, prev.Current, prev.AveragePower, prev.AverageEnergy.Total)
			}
		}
	}

	if len(update.Values) > 0 {
		err := d.comms.SendState(update)
		if err != nil {
			log.Printf("ERROR: device %s: failed to send state: %s", d.hostname, err)
		}
	}
}

func (d *ShellyPlusDevice) findSwitch(name string) *SwitchValue {
	for _, val := range d.switches {
		if val.GetName() == name {
			return val
		}
	}
	return nil
}

func (d *ShellyPlusDevice) HandleRequest(request *api.DeviceRequest) error {
	var requests []SwitchSet

	for _, req := range request.Values {
		val := d.findSwitch(req.Name)
		if val == nil {
			return fmt.Errorf("value %q not recognised", req.Name)
		}
		if req.Bool == nil {
			return fmt.Errorf("switch %q value must be a bool", req.Name)
		}
		requests = append(requests, SwitchSet{
			ID: val.ID,
			On: req.Bool.Value,
		})
	}

	d.connMu.RLock()
	defer d.connMu.RUnlock()
	if d.conn == nil {
		return fmt.Errorf("not connected to device")
	}

	for _, request := range requests {
		err := d.conn.WriteJSON(RequestFrame{
			ID:     200,
			Src:    "woodhouse",
			Method: "Switch.Set",
			Params: request,
		})
		if err != nil {
			log.Printf("ERROR: device %s: failed to set state: %s", d.hostname, err)
			return fmt.Errorf("failed to set state: %w", err)
		}
	}

	return nil
}
