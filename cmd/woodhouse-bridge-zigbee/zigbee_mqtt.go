package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-zigbee/zigbee"
	"github.com/jimjibone/woodhouse-4/wh"
)

type ZigbeeMQTT struct {
	MqttAddr        string
	RootTopic       string
	lastBackoff     time.Time
	backoffDuration time.Duration
	bridge          *wh.Bridge
	devices         map[string]*zigbee.ZigbeeDevice // devices with their IEEE address as the key.
	publish         *queue.Queue[publishMessage]
}

type publishMessage struct {
	Topic   string
	Payload []byte
}

func (zb *ZigbeeMQTT) Run(ctx context.Context, bridge *wh.Bridge) error {
	log.Printf("zigbee started")
	defer log.Printf("zigbee finished")

	zb.bridge = bridge
	zb.devices = make(map[string]*zigbee.ZigbeeDevice)
	zb.publish = queue.New[publishMessage]()
	zb.publish.Discard(true)

	for {
		client, err := zb.connect(ctx)
		if err != nil {
			return err
		}
		if client != nil {
			zb.publish.Discard(false)
			err = zb.run(ctx, client)
			zb.publish.Discard(true)
			client.Disconnect(250)
			if err != nil {
				return err
			}
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

func (zb *ZigbeeMQTT) connect(ctx context.Context) (client mqtt.Client, err error) {
	// Connect to the MQTT server.
	log.Printf("connecting to: %s", zb.MqttAddr)
	opts := mqtt.NewClientOptions().AddBroker(zb.MqttAddr).SetClientID("woodhouse-bridge-zigbee")
	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("ERROR: failed to connect to mqtt server: %s", token.Error())
		client.Disconnect(250)
		return nil, nil
	}

	// Subscribe to the zigbee root topic.
	if token := client.Subscribe(zb.RootTopic+"/#", 0, zb.messageHandler); token.Wait() && token.Error() != nil {
		log.Printf("ERROR: failed to subscribe: %s", token.Error())
		client.Disconnect(250)
		client = nil
		return nil, nil
	}

	return client, nil
}

func (zb *ZigbeeMQTT) backoff(ctx context.Context) {
	// Reset the backoff duration if the backoff has not been used for a
	// suitable amount of time.
	dt := time.Since(zb.lastBackoff)
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
}

func (zb *ZigbeeMQTT) run(ctx context.Context, client mqtt.Client) error {
	log.Printf("connected!")

	// Wait for something to publish or the context to be done.
	for {
		select {
		case <-ctx.Done():
			return nil

		case pub := <-zb.publish.Pop():
			token := client.Publish(zb.RootTopic+"/"+pub.Topic, 0, false, pub.Payload)
			if token.Wait() && token.Error() != nil {
				log.Printf("ERROR: failed to publish: %v", token.Error())
			}
		}
	}
}

func (zb *ZigbeeMQTT) messageHandler(client mqtt.Client, msg mqtt.Message) {
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) > 0 && topicParts[0] != zb.RootTopic {
		log.Printf("ERROR: received unexpected root topic: %s", msg.Topic())
		return
	}

	log.Printf("MQTT %s", msg.Topic())

	if len(topicParts) > 1 && topicParts[1] == "bridge" {

		if len(topicParts) > 2 && topicParts[2] == "state" {
			zb.handleState(string(msg.Payload()) == "online")
			return
		}

		if len(topicParts) > 2 && topicParts[2] == "devices" {
			zb.handleDeviceInfos(msg.Payload())
			return
		}

		// Ignore other bridge topics.
		return
	}

	if len(topicParts) == 2 {
		zb.handleDeviceState(topicParts[1], msg.Payload())
		return
	}

	// if len(topicParts) == 3 && (topicParts[2] == "set" || topicParts[2] == "get") {
	// 	// Ignore set and get messages.
	// 	return
	// }

	// log.Printf("ERROR: received unexpected topic: %s", msg.Topic())
}

func (zb *ZigbeeMQTT) findDeviceByName(name string) *zigbee.ZigbeeDevice {
	if name != "" {
		for _, dev := range zb.devices {
			if dev.Name() == name {
				return dev
			}
		}
	}
	return nil
}

func (zb *ZigbeeMQTT) handleState(online bool) {
	if online {
		log.Printf("zigbee2mqtt came online")
	} else {
		log.Printf("zigbee2mqtt went offline")
	}
}

func (zb *ZigbeeMQTT) handleDeviceInfos(payload []byte) {
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
			dev := zigbee.NewZigbeeDevice(zb.publishHandler)
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

func (zb *ZigbeeMQTT) handleDeviceState(friendlyName string, payload []byte) {
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

func (zb *ZigbeeMQTT) publishHandler(topic string, payload []byte) {
	zb.publish.Push(publishMessage{
		Topic:   topic,
		Payload: payload,
	})
}
