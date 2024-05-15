package shelly_v2

import (
	"fmt"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

// ShellyPlusPlugUK - device type: PlusPlugUK
type ShellyPlusPlugUK struct {
	shelly *ShellyComms
	info   *services.Info
	online *services.Online
	relay0 *services.Relay
}

func NewShellyPlusPlugUK(hostname, ip string, client *wh.Client) *ShellyPlusPlugUK {
	dev := &ShellyPlusPlugUK{
		shelly: NewShellyComms(hostname, ip, clientsapi.Device_RELAY, client),
		info:   services.NewInfo(),
		online: services.NewOnline(),
		relay0: services.NewRelay("relay0"),
	}

	initRelay(dev.relay0)

	dev.shelly.OnConnected(dev.onConnected)
	dev.shelly.OnDisconnected(dev.onDisconnected)
	dev.shelly.OnResponseFrame(dev.onResponseFrame)
	dev.shelly.OnNotificationFrame(dev.onNotificationFrame)
	dev.shelly.dev.AddService(
		dev.info,
		dev.online,
		dev.relay0,
	)
	dev.info.Name.OnAction(dev.handleNameAction)
	dev.relay0.OnAction(dev.handleRelay0Action)

	return dev
}

func (dev *ShellyPlusPlugUK) ID() string {
	return dev.shelly.ID()
}

func (dev *ShellyPlusPlugUK) Close() {
	dev.shelly.Close()
}

func (dev *ShellyPlusPlugUK) onConnected(config GetConfigResponse, status GetStatusResponse) {
	dev.info.Name.Set(config.System.Device.Name)
	dev.info.Model.Set("Shelly Plus Plug UK")
	dev.info.Manufacturer.Set("Shelly")
	dev.info.SerialNumber.Set(config.System.Device.MAC)
	dev.info.FirmwareVersion.Set(config.System.Device.FirmwareID)
	dev.info.WebUrl.Set("http://" + dev.shelly.ip)
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	// Generate inputs/scripts/switches from the config.
	for _, val := range config.Inputs {
		switch val.ID {
		default:
			dev.shelly.log.Warnf("config contained unexpected input %+v", val)
		}
	}
	for _, val := range config.Switches {
		switch val.ID {
		case 0:
			dev.relay0.SetAlias(val.Name)
		default:
			dev.shelly.log.Warnf("config contained unexpected switch %+v", val)
		}
	}

	// Update the values of inputs/scripts/switches from the status.
	for _, val := range status.Inputs {
		switch val.ID {
		default:
			dev.shelly.log.Warnf("status contained unexpected input %+v", val)
		}
	}
	for _, val := range status.Switches {
		switch val.ID {
		case 0:
			updateRelay(dev.relay0, val)
		default:
			dev.shelly.log.Warnf("status contained unexpected switch %+v", val)
		}
	}
}

func (dev *ShellyPlusPlugUK) onDisconnected() {
	dev.online.Online.Set(false)
}

func (dev *ShellyPlusPlugUK) onResponseFrame(frame ResponseFrame) {
	// dev.online.Online.Set(true)
	// dev.online.LastSeen.Set(time.Now())

	// dev.shelly.log.Warnf("response %+v", frame)

	// // Update the values of inputs/scripts/switches from the status.
	// for _, val := range frame.NotifyStatus.Inputs {
	// 	switch val.ID {
	// 	default:
	// 		dev.shelly.log.Warnf("status contained unexpected input %+v", val)
	// 	}
	// }
	// for _, val := range frame.NotifyStatus.Switches {
	// 	switch val.ID {
	// 	case 0:
	// 		updateRelay(dev.relay0, val)
	// 	default:
	// 		dev.shelly.log.Warnf("status contained unexpected switch %+v", val)
	// 	}
	// }
}

func (dev *ShellyPlusPlugUK) onNotificationFrame(frame NotificationFrame) {
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	// dev.shelly.log.Warnf("notification %+v", frame)

	// Update the values of inputs/scripts/switches from the status.
	for _, val := range frame.NotifyStatus.Inputs {
		switch val.ID {
		default:
			dev.shelly.log.Warnf("status contained unexpected input %+v", val)
		}
	}
	for _, val := range frame.NotifyStatus.Switches {
		switch val.ID {
		case 0:
			updateRelay(dev.relay0, val)
		default:
			dev.shelly.log.Warnf("status contained unexpected switch %+v", val)
		}
	}
}

func (dev *ShellyPlusPlugUK) handleNameAction(val string) {
	dev.shelly.log.Errorf("not changing name to %s", val)
}

func (dev *ShellyPlusPlugUK) handleRelay0Action(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	var switchSet SwitchSet

	for _, req := range request.Values {
		switch req.Id {
		case dev.relay0.On.ID():
			if req.GetBool() == nil {
				return services.ErrIncorrectTypeFor(dev.relay0.On)
			}
			switchSet = SwitchSet{
				ID: 0,
				On: req.GetBool().GetValue(),
			}
		}
	}

	err := dev.shelly.RequestSwitchSet(request.ActionId, switchSet)
	if err != nil {
		dev.shelly.log.Errorf("failed to set switch: %s", err)
		return fmt.Errorf("failed to set switch: %w", err)
	}

	return nil
}
