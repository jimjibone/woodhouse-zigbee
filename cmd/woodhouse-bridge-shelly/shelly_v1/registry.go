package shelly_v1

import "github.com/jimjibone/woodhouse-4/wh/v1/devices"

type Device interface {
	ID() string
	Device() devices.Device
	Close()
}

type DeviceGenerator func(hostname, ip string) Device

var registry = make(map[string]DeviceGenerator)

func registerDevice(deviceType string, generator DeviceGenerator) {
	registry[deviceType] = generator
}

func Generate(deviceType, hostname, ip string) Device {
	if gen, found := registry[deviceType]; found {
		return gen(hostname, ip)
	}
	return nil
}
