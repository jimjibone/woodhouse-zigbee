package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-zigbee/zigbee"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"github.com/jimjibone/woodhouse-4/wh/v1"
)

const (
	minBackoff = time.Second
	maxBackoff = 30 * time.Second
)

type ZigbeeWebsockets struct {
	log             *log.Context
	FS              stores.Store
	WebAddr         string
	WsAddr          string
	RootTopic       string
	lastBackoff     time.Time
	lastRestart     time.Time
	backoffDuration time.Duration
	client          *wh.Client
	connMu          sync.RWMutex
	conn            *websocket.Conn
	devices         map[string]zigbee.ZigbeeDevice // devices with their IEEE address as the key.
}

func (zb *ZigbeeWebsockets) Run(ctx context.Context, client *wh.Client) error {
	if zb.log == nil {
		zb.log = log.NewContext(log.DefaultLogger, "zigbee", log.DebugLevel)
	}

	log.Infof("started")
	defer log.Infof("finished")

	zb.client = client
	zb.devices = make(map[string]zigbee.ZigbeeDevice)

	for {
		// Try to connect.
		zb.log.Infof("creating websocket")
		conn := zb.connect()
		if conn != nil {
			zb.connMu.Lock()
			zb.conn = conn
			zb.connMu.Unlock()

			// Close the websocket when the context is cancelled.
			go func() {
				<-ctx.Done()
				zb.log.Infof("closing websocket")
				zb.connMu.Lock()
				if zb.conn != nil {
					zb.conn.Close()
					zb.conn = nil
				}
				zb.connMu.Unlock()
			}()

			// Receive updates and send requests.
			zb.recv(ctx, conn)
		}

		// Check if we're done.
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Backoff for a short while before the next connection attempt.
		zb.backoff(ctx)
	}
}

func (zb *ZigbeeWebsockets) connect() *websocket.Conn {
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/api", zb.WsAddr), nil)
	if err != nil {
		zb.log.Errorln("dial:", err)
		return nil
	}
	return conn
}

func (zb *ZigbeeWebsockets) recv(ctx context.Context, conn *websocket.Conn) {
	saveJson := func(filename string, data []byte) {
		var tmp interface{}
		err := json.Unmarshal(data, &tmp)
		if err != nil {
			panic(err)
		}
		payload, err := json.MarshalIndent(tmp, "", "    ")
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(filename, payload, 0644)
		if err != nil {
			panic(err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			zb.log.Errorf("read: %s", err)
			conn.Close()
			return
		}

		frame := struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}{}
		err = json.Unmarshal(message, &frame)
		if err != nil {
			zb.log.Errorln("unmarshal:", err)
			conn.Close()
			return
		}

		switch frame.Topic {
		case "bridge/info":
			saveJson("zigbee-bridge-info.json", frame.Payload)

		case "bridge/devices":
			saveJson("zigbee-bridge-devices.json", frame.Payload)
			zb.handleDeviceInfos(frame.Payload)

		case "bridge/config":
			saveJson("zigbee-bridge-config.json", frame.Payload)

		case "bridge/state", "bridge/groups", "bridge/extensions", "bridge/logging", "bridge/log":
			// Ignore these.

		default:
			// zb.log.Debugf("----> recv: %s", message)
			if strings.HasSuffix(frame.Topic, "/availability") {
				zb.handleDeviceAvailability(strings.TrimSuffix(frame.Topic, "/availability"), frame.Payload)
			} else {
				zb.handleDeviceState(frame.Topic, frame.Payload)
			}
		}
	}
}

func (zb *ZigbeeWebsockets) backoff(ctx context.Context) {
	// Reset the backoff duration if the backoff has not been used for a
	// suitable amount of time.
	dt := time.Since(zb.lastRestart)
	if dt > zb.backoffDuration {
		zb.log.Debugf("backoff reset after %s", dt)
		zb.backoffDuration = minBackoff
	}
	zb.lastBackoff = time.Now()
	zb.log.Infof("starting backoff for %s", zb.backoffDuration)
	timer := time.NewTimer(zb.backoffDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	zb.log.Debugf("backoff finished")
	zb.backoffDuration = zb.backoffDuration * 2
	if zb.backoffDuration > maxBackoff {
		zb.backoffDuration = maxBackoff
	}
	zb.lastRestart = time.Now()
}

func (zb *ZigbeeWebsockets) handleDeviceInfos(payload []byte) {
	// zb.log.Printf("device infos: %s", payload)

	if zb.FS != nil {
		zb.FS.Set("device_infos.json", payload)
	}

	var devices []zigbee.DeviceInfo
	if err := json.Unmarshal(payload, &devices); err != nil {
		zb.log.Errorf("failed to unmarshal device infos: %v", err)
		return
	}

	zb.log.Debugf("device infos: %d", len(devices))
	for i, dev := range devices {
		fmt.Printf("  %d: %+v\n", i, dev)
	}

	// Update or create devices.
	for _, info := range devices {
		if dev, found := zb.devices[info.IEEEAddress]; found {
			dev.UpdateInfo(info)
		} else {
			dev := zigbee.GenerateDevice(info, zb.client, zb.WebAddr, zb.requestHandler)
			if dev != nil {
				zb.devices[info.IEEEAddress] = dev
			}
		}
	}
}

func (zb *ZigbeeWebsockets) handleDeviceAvailability(friendlyName string, payload []byte) {
	availability := struct {
		State string `json:"state"`
	}{}
	err := json.Unmarshal(payload, &availability)
	if err != nil {
		zb.log.Errorf("failed to unmarshal device availability: name: %q, payload: %s, err: %s", friendlyName, payload, err)
		return
	}

	// Update device.
	if dev := zb.findDeviceByName(friendlyName); dev != nil {
		switch availability.State {
		case "online":
			dev.UpdateOnline(true)
		case "offline":
			dev.UpdateOnline(false)
		default:
			zb.log.Errorf("received unexpected device availability for unknown device: %q %q", friendlyName, availability.State)
		}
	} else {
		zb.log.Errorf("received device availability for unknown device: %q %q", friendlyName, availability.State)
	}
}

func (zb *ZigbeeWebsockets) handleDeviceState(friendlyName string, payload []byte) {
	// zb.log.Printf("device state: %s: %s", friendlyName, payload)
	if len(payload) == 0 {
		return
	}
	if bytes.Equal(payload, []byte(`""`)) {
		return
	}

	var state zigbee.DeviceState
	if err := json.Unmarshal(payload, &state); err != nil {
		zb.log.Errorf("failed to unmarshal device state - error: %v", err)
		zb.log.Errorf("failed to unmarshal device state - friendlyName: %q, payload: %s", friendlyName, payload)
		return
	}

	// zb.log.Printf("device state: %s\n%s", friendlyName, payload)
	zb.log.Debugf("device state: %s\n%s", friendlyName, state.String())

	// Update and possibly add devices.
	if dev := zb.findDeviceByName(friendlyName); dev != nil {
		dev.UpdateState(state)
	}
	//else {
	//	zb.log.Errorf("received device state for unknown device: %s\n%s", friendlyName, state.String())
	//}
	//}
}

func (zb *ZigbeeWebsockets) findDeviceByName(name string) zigbee.ZigbeeDevice {
	if name != "" {
		for _, dev := range zb.devices {
			if dev.Name() == name {
				return dev
			}
		}
	}
	return nil
}

func (zb *ZigbeeWebsockets) requestHandler(request zigbee.ZigbeeRequest) {
	zb.connMu.Lock()
	defer zb.connMu.Unlock()
	if zb.conn == nil {
		zb.log.Warnf("not connected to z2m for request %q", request.Topic)
		return
	}
	zb.log.Debugf("sending %q: %s", request.Topic, request.Payload)
	err := zb.conn.WriteJSON(struct {
		Topic   string          `json:"topic"`
		Payload json.RawMessage `json:"payload"`
	}{
		Topic:   request.Topic,
		Payload: request.Payload,
	})
	if err != nil {
		zb.log.Errorf("failed to send request: %s", err)
	}
}
