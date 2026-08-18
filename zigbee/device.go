package zigbee

import (
	"github.com/jimjibone/log"
	"github.com/jimjibone/wh/v1/bridges"
)

type ZigbeeDevice interface {
	Name() string
	Device() *bridges.Device
	UpdateOnline(bool)
	UpdateInfo(DeviceInfo)
	UpdateState(DeviceState)
}

type ZigbeeRequest struct {
	Topic   string
	Payload []byte
}

func GenerateDevice(info DeviceInfo, client *bridges.Bridge, baseUrl string, requests func(ZigbeeRequest), correlator *ResponseCorrelator) ZigbeeDevice {
	if info.Type == "Coordinator" {
		return nil
	}
	if !info.InterviewCompleted {
		return nil
	}
	if len(info.Definition.Exposes) == 0 {
		return nil
	}

	firstExpose := info.Definition.Exposes[0]
	switch firstExpose.Type {
	// case "light":
	// 	return NewZigbeeLight(info, client, baseUrl, requests)

	// // case "switch":

	// case "climate":
	// 	return NewZigbeeClimate(info, client, baseUrl, requests)

	default:
		dev := NewZigbeeDeviceImpl(info, client, baseUrl, requests, correlator)
		if dev != nil {
			return dev
		}
	}

	log.Errorf("unsupported first exposed type: %s", firstExpose.Type)
	log.Errorf("unsupported info: %q %s - exposes: %d, options: %d", info.FriendlyName, info.IEEEAddress, len(info.Definition.Exposes), len(info.Definition.Options))
	for i, v := range info.Definition.Exposes {
		log.Errorf("  expose %2d: %s", i, v)
	}
	for i, v := range info.Definition.Options {
		log.Errorf("  option %2d: %s", i, v)
	}
	return nil
}
