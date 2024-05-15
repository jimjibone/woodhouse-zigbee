package shelly_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
)

const (
	minBackoff = 1 * time.Second
	maxBackoff = 10 * time.Second
)

// ShellyComms manages the lifecycle of device comms.
type ShellyComms struct {
	log   *log.Context
	close func()
	wg    sync.WaitGroup

	client *wh.Client
	dev    *devices.DeviceImpl
	added  bool

	ip     string
	connMu sync.RWMutex
	conn   *websocket.Conn

	lastBackoff     time.Time
	lastRestart     time.Time
	backoffDuration time.Duration

	onConnected         func(config GetConfigResponse, status GetStatusResponse)
	onDisconnected      func()
	onResponseFrame     func(ResponseFrame)
	onNotificationFrame func(NotificationFrame)
}

func NewShellyComms(hostname, ip string, typ clientsapi.Device_DeviceType, client *wh.Client) *ShellyComms {
	ctx, cancel := context.WithCancel(context.Background())
	dev := &ShellyComms{
		log:    log.NewContext(log.DefaultLogger, hostname, log.DebugLevel),
		close:  cancel,
		client: client,
		dev:    devices.NewDevice(hostname, typ),
		ip:     ip,
	}
	dev.wg.Add(1)
	go dev.run(ctx)
	return dev
}

func (dev *ShellyComms) Close() {
	dev.close()
	dev.wg.Wait()
}

func (dev *ShellyComms) ID() string {
	return dev.dev.ID()
}

func (dev *ShellyComms) OnConnected(handler func(config GetConfigResponse, status GetStatusResponse)) {
	dev.onConnected = handler
}

func (dev *ShellyComms) OnDisconnected(handler func()) {
	dev.onDisconnected = handler
}

func (dev *ShellyComms) OnResponseFrame(handler func(ResponseFrame)) {
	dev.onResponseFrame = handler
}

func (dev *ShellyComms) OnNotificationFrame(handler func(NotificationFrame)) {
	dev.onNotificationFrame = handler
}

func (dev *ShellyComms) RequestSwitchSet(id string, request SwitchSet) error {
	dev.connMu.RLock()
	defer dev.connMu.RUnlock()
	if dev.conn == nil {
		return fmt.Errorf("not connected to device")
	}

	err := dev.conn.WriteJSON(RequestFrame{
		ID:     FrameID(id),
		Src:    "woodhouse-4",
		Method: "Switch.Set",
		Params: request,
	})
	if err != nil {
		return err
	}

	return nil
}

func (dev *ShellyComms) run(ctx context.Context) {
	defer dev.wg.Done()

	dev.log.Infof("started")
	defer dev.log.Infof("finished")

	// Close the websocket when the context is cancelle
	go func() {
		<-ctx.Done()
		dev.disconnect()
	}()

	for {
		// Try to connect.
		err := dev.connect()
		if err != nil {
			dev.log.Errorf("failed to connect: %s", err)
		} else {
			// Receive updates and send requests.
			dev.recv(ctx)

			// Close teh websocket.
			dev.disconnect()
		}

		// Check if we're done.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Backoff for a short while before the next connection attempt.
		dev.backoff(ctx)
	}
}

func (dev *ShellyComms) connect() (err error) {
	var conn *websocket.Conn
	conn, _, err = websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/rpc", dev.ip), nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	// Close the conn on error.
	defer func() {
		if err != nil {
			conn.Close()
			conn = nil
		}
	}()

	dev.log.Debugf("created websocket")

	err = conn.WriteJSON(RequestFrame{
		ID:     "120",
		Src:    "woodhouse-4",
		Method: "Shelly.GetConfig",
	})
	// err = conn.WriteMessage(websocket.TextMessage, []byte(`{"id":300, "src":"woodhouse", "method":"Shelly.GetConfig"}`))
	if err != nil {
		return fmt.Errorf("write get config: %w", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read get config: %w", err)
	}
	// dev.log.Debugf("--> recv: %s", message)

	var frame ResponseFrame
	err = json.Unmarshal(message, &frame)
	if err != nil {
		return fmt.Errorf("unmarshal config frame: %w", err)
	}
	// dev.log.Debugf("config frame: %+v", frame)

	var configResponse GetConfigResponse
	if frame.Result != nil {
		err = json.Unmarshal(frame.Result, &configResponse)
		if err != nil {
			return fmt.Errorf("unmarshal config: %w", err)
		}
		dev.log.Debugf("config: %+v", configResponse)
	}

	err = conn.WriteJSON(RequestFrame{
		ID:     "121",
		Src:    "woodhouse-4",
		Method: "Shelly.GetStatus",
	})
	// err = conn.WriteMessage(websocket.TextMessage, []byte(`{"id":301, "src":"woodhouse", "method":"Shelly.GetStatus"}`))
	if err != nil {
		return fmt.Errorf("write get status: %w", err)
	}

	_, message, err = conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read get status: %w", err)
	}
	// dev.log.Debugf("--> recv: %s", message)

	err = json.Unmarshal(message, &frame)
	if err != nil {
		return fmt.Errorf("unmarshal status frame: %w", err)
	}
	// dev.log.Debugf("status frame: %+v", frame)

	var statusResponse GetStatusResponse
	if frame.Result != nil {
		err = json.Unmarshal(frame.Result, &statusResponse)
		if err != nil {
			return fmt.Errorf("unmarshal status: %w", err)
		}
		dev.log.Debugf("status: %+v", statusResponse)
	}

	dev.connMu.Lock()
	dev.conn = conn
	dev.connMu.Unlock()

	dev.logDeviceInfo(configResponse, statusResponse)

	// Add the device to the client once we've fully connected to it.
	if !dev.added {
		dev.added = true
		dev.client.AddDevice(dev.dev)
	}

	if dev.onConnected != nil {
		dev.onConnected(configResponse, statusResponse)
	}

	return nil
}

func (dev *ShellyComms) disconnect() {
	dev.connMu.Lock()
	defer dev.connMu.Unlock()
	if dev.conn != nil {
		dev.log.Infof("closing websocket")
		if dev.onDisconnected != nil {
			dev.onDisconnected()
		}
		dev.conn.Close()
		dev.conn = nil
	}
}

func (dev *ShellyComms) recv(ctx context.Context) {
	dev.log.Debugf("recv started")
	defer dev.log.Debugf("recv finished")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := dev.conn.ReadMessage()
		if err != nil {
			dev.log.Errorf("read: %s", err)
			return
		}
		// dev.log.Infof("--> recv: %s", message)

		switch DetectFrameType(message) {
		default:
			dev.log.Warnf("unknown frame type for message: %s", message)

		case ResponseFrameType:
			var frame ResponseFrame
			err = json.Unmarshal(message, &frame)
			if err != nil {
				dev.log.Warnf("unmarshal: %s, from message: %s", err, message)
			} else {
				dev.log.Debugf("--> recv response frame: %+v", frame)
				if dev.onResponseFrame != nil {
					dev.onResponseFrame(frame)
				}
			}

		case NotificationFrameType:
			var frame NotificationFrame
			err = json.Unmarshal(message, &frame)
			if err != nil {
				dev.log.Warnf("unmarshal: %s, from message: %s", err, message)
			} else {
				dev.log.Debugf("--> recv notification frame: %+v", frame)
				if frame.NotifyStatus != nil {
					if dev.onNotificationFrame != nil {
						dev.onNotificationFrame(frame)
					}
				}
			}
		}
	}
}

func (dev *ShellyComms) backoff(ctx context.Context) {
	// Reset the backoff duration if the backoff has not been used for a
	// suitable amount of time.
	dt := time.Since(dev.lastRestart)
	if dt > dev.backoffDuration {
		dev.log.Infof("backoff reset after %s", dt)
		dev.backoffDuration = minBackoff
	}
	dev.lastBackoff = time.Now()
	dev.log.Infof("starting backoff for %s", dev.backoffDuration)
	timer := time.NewTimer(dev.backoffDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	dev.log.Infof("backoff finished")
	dev.backoffDuration = dev.backoffDuration * 2
	if dev.backoffDuration > maxBackoff {
		dev.backoffDuration = maxBackoff
	}
	dev.lastRestart = time.Now()
}

func (dev *ShellyComms) logDeviceInfo(config GetConfigResponse, status GetStatusResponse) {
	name := config.System.Device.Name
	inputs := make(map[int]*InputValue)
	scripts := make(map[int]*ScriptValue)
	switches := make(map[int]*SwitchValue)

	// Get the inputs/scripts/switches from the config.
	for _, val := range config.Inputs {
		inputs[val.ID] = &InputValue{
			ID:     val.ID,
			Name:   val.Name,
			Type:   val.Type,
			Invert: val.Invert,
		}
	}
	for _, val := range config.Scripts {
		scripts[val.ID] = &ScriptValue{
			ID:     val.ID,
			Name:   val.Name,
			Enable: val.Enable,
		}
	}
	for _, val := range config.Switches {
		switches[val.ID] = &SwitchValue{
			ID:   val.ID,
			Name: val.Name,
		}
	}

	// Get values from the status.
	for _, val := range status.Inputs {
		if prev, found := inputs[val.ID]; found {
			prev.Timestamp = time.Now()
			prev.State = val.State
		}
	}
	for _, val := range status.Scripts {
		if prev, found := scripts[val.ID]; found {
			prev.Timestamp = time.Now()
			prev.Running = val.Running
		}
	}
	for _, val := range status.Switches {
		if prev, found := switches[val.ID]; found {
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

	dev.log.Infof("device name: %s", name)

	var ids []int
	for _, val := range inputs {
		ids = append(ids, val.ID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		val := inputs[id]
		dev.log.Infof("input id: %d, name: %q, type: %s, invert: %t, ts: %s, state: %t", id, val.Name, val.Type, val.Invert, val.Timestamp.Format("2006/01/02 15:04:05"), val.State)
	}

	ids = nil
	for _, val := range scripts {
		ids = append(ids, val.ID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		val := scripts[id]
		dev.log.Infof("script id: %d, name: %q, ts: %s, enable: %t, running %t", id, val.Name, val.Timestamp.Format("2006/01/02 15:04:05"), val.Enable, val.Running)
	}

	ids = nil
	for _, val := range switches {
		ids = append(ids, val.ID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		val := switches[id]
		dev.log.Infof("switch id: %d, name: %q, ts: %s, state: %t, temp: %0.1f °C, voltage: %.1f V, current: %.1f A, avg.power: %.1f W, avg.energy.total: %.3f Wh", id, val.Name, val.Timestamp.Format("2006/01/02 15:04:05"), val.State, val.Temperature, val.Voltage, val.Current, val.AveragePower, val.AverageEnergy.Total)
	}
}
