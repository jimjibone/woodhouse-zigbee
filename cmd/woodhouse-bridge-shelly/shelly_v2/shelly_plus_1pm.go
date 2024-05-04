package shelly_v2

import (
	"fmt"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

// ShellyPlus1PM - device type: Plus1PM
type ShellyPlus1PM struct {
	shelly  *ShellyComms
	info    *services.Info
	online  *services.Online
	switch0 *services.Switch
	relay0  *services.Relay
}

func NewShellyPlus1PM(hostname, ip string, client *wh.Client) *ShellyPlus1PM {
	dev := &ShellyPlus1PM{
		shelly:  NewShellyComms(hostname, ip, clientsapi.Device_RELAY, client),
		info:    services.NewInfo(),
		online:  services.NewOnline(),
		switch0: services.NewSwitchID("switch0"),
		relay0:  services.NewRelayID("relay0"),
	}
	dev.shelly.OnConnected(dev.onConnected)
	dev.shelly.OnDisconnected(dev.onDisconnected)
	dev.shelly.OnResponseFrame(dev.onResponseFrame)
	dev.shelly.OnNotificationFrame(dev.onNotificationFrame)
	dev.shelly.dev.AddService(
		dev.info,
		dev.online,
		dev.switch0,
		dev.relay0,
	)
	dev.info.Name.OnAction(dev.handleNameAction)
	dev.relay0.OnAction(dev.handleRelayAction)

	return dev
}

func (dev *ShellyPlus1PM) ID() string {
	return dev.shelly.ID()
}

func (dev *ShellyPlus1PM) Close() {
	dev.shelly.Close()
}

func (dev *ShellyPlus1PM) onConnected(config GetConfigResponse, status GetStatusResponse) {
	dev.info.Name.Set(config.System.Device.Name)
	dev.info.Model.Set("Shelly Plus 1PM")
	dev.info.Manufacturer.Set("Shelly")
	dev.info.SerialNumber.Set(config.System.Device.MAC)
	dev.info.FirmwareVersion.Set(config.System.Device.FirmwareID)
	dev.info.WebUrl.Set("http://" + dev.shelly.ip)
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	// Generate inputs/scripts/switches from the config.
	for _, val := range config.Inputs {
		if val.ID == 0 {
			dev.switch0.SetAlias(val.Name)
		} else {
			dev.shelly.log.Warnf("config contained unexpected input %+v", val)
		}
	}
	for _, val := range config.Switches {
		if val.ID == 0 {
			dev.relay0.SetAlias(val.Name)
		} else {
			dev.shelly.log.Warnf("config contained unexpected switch %+v", val)
		}
	}

	// Update the values of inputs/scripts/switches from the status.
	for _, val := range status.Inputs {
		if val.ID == 0 {
			dev.switch0.On.Set(val.State)
		} else {
			dev.shelly.log.Warnf("status contained unexpected input %+v", val)
		}
	}
	for _, val := range status.Switches {
		if val.ID == 0 {
			if val.Output != nil {
				dev.relay0.On.Set(*val.Output)
			}
			if val.AveragePower != nil {
				dev.relay0.Power.Set(*val.AveragePower)
			}
			if val.Voltage != nil {
				dev.relay0.Voltage.Set(*val.Voltage)
			}
			if val.Current != nil {
				dev.relay0.Current.Set(*val.Current)
			}
			if val.Temperature != nil {
				dev.relay0.Temperature.Set(val.Temperature.Centigrade)
			}
		} else {
			dev.shelly.log.Warnf("status contained unexpected switch %+v", val)
		}
	}
}

func (dev *ShellyPlus1PM) onDisconnected() {
	dev.online.Online.Set(false)
}

func (dev *ShellyPlus1PM) onResponseFrame(ResponseFrame) {

}

func (dev *ShellyPlus1PM) onNotificationFrame(frame NotificationFrame) {
	dev.online.Online.Set(true)
	dev.online.LastSeen.Set(time.Now())

	// Update the values of inputs/scripts/switches from the status.
	for _, val := range frame.NotifyStatus.Inputs {
		if val.ID == 0 {
			dev.switch0.On.Set(val.State)
		} else {
			dev.shelly.log.Warnf("status contained unexpected input %+v", val)
		}
	}
	for _, val := range frame.NotifyStatus.Switches {
		if val.ID == 0 {
			if val.Output != nil {
				dev.relay0.On.Set(*val.Output)
			}
			if val.AveragePower != nil {
				dev.relay0.Power.Set(*val.AveragePower)
			}
			if val.Voltage != nil {
				dev.relay0.Voltage.Set(*val.Voltage)
			}
			if val.Current != nil {
				dev.relay0.Current.Set(*val.Current)
			}
			if val.Temperature != nil {
				dev.relay0.Temperature.Set(val.Temperature.Centigrade)
			}
		} else {
			dev.shelly.log.Warnf("status contained unexpected switch %+v", val)
		}
	}
}

func (dev *ShellyPlus1PM) handleNameAction(val string) {
	dev.shelly.log.Errorf("not changing name to %s", val)
}

func (dev *ShellyPlus1PM) handleRelayAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
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

	err := dev.shelly.RequestSwitchSet(switchSet)
	if err != nil {
		dev.shelly.log.Errorf("failed to set switch: %s", err)
		return fmt.Errorf("failed to set switch: %w", err)
	}

	return nil
}
