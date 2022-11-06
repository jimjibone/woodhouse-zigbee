package shelly_v1

import "github.com/jimjibone/woodhouse-4/wh"

type Device interface {
	wh.Device
	UpdateInfo()                 // Update the device info.
	UpdateState(fullUpdate bool) // Update the device state.
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
