package shelly_v2

import (
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/attributes"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
)

func initRelay(relay *services.Relay) {
	relay.On.SetOptional(attributes.Required)
	relay.Voltage.SetOptional(attributes.Required)
	relay.Current.SetOptional(attributes.Required)
	relay.Power.SetOptional(attributes.Required)
	relay.Temperature.SetOptional(attributes.Required)
}

func updateRelay(relay *services.Relay, val GetStatusResponseSwitch) {
	if val.Output != nil {
		relay.On.Set(*val.Output)
	}
	if val.AveragePower != nil {
		relay.Power.Set(*val.AveragePower)
	}
	if val.Voltage != nil {
		relay.Voltage.Set(*val.Voltage)
	}
	if val.Current != nil {
		relay.Current.Set(*val.Current)
	}
	if val.Temperature != nil {
		relay.Temperature.Set(val.Temperature.Centigrade)
	}
}
