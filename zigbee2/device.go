package zigbee

import (
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
)

type ZigbeeDevice interface {
	Name() string
	Device() *devices.Device
	UpdateOnline(bool)
	UpdateInfo(DeviceInfo)
	UpdateState(DeviceState)
}

type ZigbeeRequest struct {
	Topic   string
	Payload []byte
}

func GenerateDevice(info DeviceInfo, client *wh.Client, baseUrl string, requests func(ZigbeeRequest)) ZigbeeDevice {
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
	case "light":
		return NewZigbeeLight(info, client, baseUrl, requests)

	case "switch":
		log.Errorf("unsupported first exposed type: %s", firstExpose.Type)

	case "climate":
		return NewZigbeeClimate(info, client, baseUrl, requests)

	default:
		log.Errorf("unsupported first exposed type: %s", firstExpose.Type)
	}
	return nil
}
