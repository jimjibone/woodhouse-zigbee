package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-zigbee/zigbee"
	"github.com/jimjibone/woodhouse-4/wh"
)

const (
	minBackoff = time.Second
	maxBackoff = 30 * time.Second
)

type ZigbeeWebsockets struct {
	WebAddr         string
	WsAddr          string
	RootTopic       string
	lastBackoff     time.Time
	lastRestart     time.Time
	backoffDuration time.Duration
	bridge          *wh.Bridge
	connMu          sync.RWMutex
	conn            *websocket.Conn
	devices         map[string]*zigbee.ZigbeeDevice // devices with their IEEE address as the key.
}

func (zb *ZigbeeWebsockets) Run(ctx context.Context, bridge *wh.Bridge) error {
	log.Printf("zigbee started")
	defer log.Printf("zigbee finished")

	zb.bridge = bridge
	zb.devices = make(map[string]*zigbee.ZigbeeDevice)

	for {
		// Try to connect.
		log.Printf("creating websocket")
		conn := zb.connect()
		if conn != nil {
			zb.connMu.Lock()
			zb.conn = conn
			zb.connMu.Unlock()

			// Close the websocket when the context is cancelled.
			go func() {
				<-ctx.Done()
				log.Printf("closing websocket")
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
		log.Println("ERROR: dial:", err)
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
			log.Printf("ERROR: read: %s", err)
			conn.Close()
			return
		}

		frame := struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}{}
		err = json.Unmarshal(message, &frame)
		if err != nil {
			log.Println("ERROR: unmarshal:", err)
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
			// log.Printf("----> recv: %s", message)
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
		log.Printf("backoff reset after %s", dt)
		zb.backoffDuration = minBackoff
	}
	zb.lastBackoff = time.Now()
	log.Printf("starting backoff for %s", zb.backoffDuration)
	timer := time.NewTimer(zb.backoffDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	log.Printf("backoff finished")
	zb.backoffDuration = zb.backoffDuration * 2
	if zb.backoffDuration > maxBackoff {
		zb.backoffDuration = maxBackoff
	}
	zb.lastRestart = time.Now()
}

func (zb *ZigbeeWebsockets) handleDeviceInfos(payload []byte) {
	// log.Printf("device infos: %s", payload)

	var devices []zigbee.DeviceInfo
	if err := json.Unmarshal(payload, &devices); err != nil {
		log.Printf("ERROR: failed to unmarshal device infos: %v", err)
		return
	}

	log.Printf("device infos: %d", len(devices))
	for i, dev := range devices {
		fmt.Printf("  %d: %+v\n", i, dev)
	}

	// Update or create devices.
	for _, info := range devices {
		if dev, found := zb.devices[info.IEEEAddress]; found {
			dev.UpdateInfo(info)
		} else {
			dev := zigbee.NewZigbeeDevice(zb.WebAddr, zb.requestHandler)
			dev.UpdateInfo(info)
			if err := json.Unmarshal(payload, &devices); err != nil {
				log.Printf("ERROR: failed to update device %q info: %v", info.FriendlyName, err)
				continue
			}
			zb.devices[info.IEEEAddress] = dev
			// zb.bridge.AddDevice(dev.ID(), dev)
		}
	}
}

func (zb *ZigbeeWebsockets) handleDeviceAvailability(friendlyName string, payload []byte) {
	// Update and possibly add devices.
	if dev := zb.findDeviceByName(friendlyName); dev != nil {
		switch string(payload) {
		case `"online"`:
			dev.UpdateOnline(true)
		case `"offline"`:
			dev.UpdateOnline(false)
		default:
			log.Printf("ERROR: received unexpected device availability for unknown device: %q %s", friendlyName, payload)
		}
	} else {
		log.Printf("ERROR: received device availability for unknown device: %q %s", friendlyName, payload)
	}
}

func (zb *ZigbeeWebsockets) handleDeviceState(friendlyName string, payload []byte) {
	// log.Printf("device state: %s: %s", friendlyName, payload)

	var state zigbee.DeviceState
	if err := json.Unmarshal(payload, &state); err != nil {
		log.Printf("ERROR: failed to unmarshal device state: %v", err)
		return
	}

	// log.Printf("device state: %s\n%s", friendlyName, payload)
	log.Printf("device state: %s\n%s", friendlyName, state.String())

	// Update and possibly add devices.
	if dev := zb.findDeviceByName(friendlyName); dev != nil {
		dev.UpdateState(state)

		if !dev.Added {
			// Add the device to the bridge now as we should now have its info
			// and state.
			dev.Added = true
			zb.bridge.AddDevice(dev.ID(), dev)
		}
	} else {
		log.Printf("ERROR: received device state for unknown device: %s\n%s", friendlyName, state.String())
	}
}

func (zb *ZigbeeWebsockets) findDeviceByName(name string) *zigbee.ZigbeeDevice {
	if name != "" {
		for _, dev := range zb.devices {
			if dev.Name() == name {
				return dev
			}
		}
	}
	return nil
}

func (zb *ZigbeeWebsockets) requestHandler(topic string, payload []byte) {
	zb.connMu.RLock()
	defer zb.connMu.RUnlock()
	if zb.conn == nil {
		log.Printf("WARN: not connected to z2m for request %q", topic)
		return
	}
	log.Printf("sending %s: %s", topic, payload)
	err := zb.conn.WriteJSON(struct {
		Topic   string          `json:"topic"`
		Payload json.RawMessage `json:"payload"`
	}{
		Topic:   topic,
		Payload: payload,
	})
	if err != nil {
		log.Printf("ERROR: failed to send request: %s", err)
	}
}
