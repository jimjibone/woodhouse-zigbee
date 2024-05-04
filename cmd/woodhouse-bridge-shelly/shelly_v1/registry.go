package shelly_v1

import (
	"github.com/jimjibone/woodhouse-4/wh/v1"
)

type Device interface {
	ID() string
	Close()
}

type DeviceGenerator func(hostname, ip string, client *wh.Client) Device

var registry = make(map[string]DeviceGenerator)

func registerDevice(deviceType string, generator DeviceGenerator) {
	registry[deviceType] = generator
}

func Generate(deviceType, hostname, ip string, client *wh.Client) Device {
	if gen, found := registry[deviceType]; found {
		return gen(hostname, ip, client)
	}
	return nil
}
