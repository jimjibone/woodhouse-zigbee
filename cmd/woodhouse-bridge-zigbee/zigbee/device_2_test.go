package zigbee_test

import (
	"encoding/json"
	"testing"

	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-zigbee/zigbee"
)

func TestDevice2(t *testing.T) {
	// Load device info.
	var info zigbee.DeviceInfo
	err := json.Unmarshal(Device2Info, &info)
	if err != nil {
		t.Fatalf("failed to unmarshal device 2 info: %v", err)
	}
	t.Logf("device 2 info: %+v", info)

	dev := zigbee.NewZigbeeDevice("", nil)
	err = dev.UpdateInfo(info)
	if err != nil {
		t.Fatalf("failed to update device 2 info: %v", err)
	}
	t.Logf("device 2: %+v", dev)

	// Load device state.
	var state zigbee.DeviceState
	err = json.Unmarshal(Device2State, &state)
	if err != nil {
		t.Fatalf("failed to unmarshal device 2 state: %v", err)
	}
	t.Logf("device 2 state: %+v", state)

	err = dev.UpdateState(state)
	if err != nil {
		t.Fatalf("failed to update device info: %v", err)
	}
	t.Logf("device 2: %+v", dev)
}

var Device2State = []byte(`{
	"battery": 46,
	"brightness": 255,
	"counter": 1,
	"last_seen": "2022-11-19T23:46:29.668Z",
	"linkquality": 111,
	"update": {
		"state": "idle"
	},
	"update_available": false
}`)

var Device2Info = []byte(`{
	"date_code": "20190410",
	"definition": {
		"description": "Hue dimmer switch",
		"exposes": [
			{
				"access": 1,
				"description": "Remaining battery in %",
				"name": "battery",
				"property": "battery",
				"type": "numeric",
				"unit": "%",
				"value_max": 100,
				"value_min": 0
			},
			{
				"access": 1,
				"description": "Triggered action (e.g. a button click)",
				"name": "action",
				"property": "action",
				"type": "enum",
				"values": [
					"on_press",
					"on_hold",
					"on_hold_release",
					"up_press",
					"up_hold",
					"up_hold_release",
					"down_press",
					"down_hold",
					"down_hold_release",
					"off_press",
					"off_hold",
					"off_hold_release"
				]
			},
			{
				"access": 1,
				"name": "action_duration",
				"property": "action_duration",
				"type": "numeric",
				"unit": "second"
			},
			{
				"access": 1,
				"description": "Link quality (signal strength)",
				"name": "linkquality",
				"property": "linkquality",
				"type": "numeric",
				"unit": "lqi",
				"value_max": 255,
				"value_min": 0
			}
		],
		"model": "324131092621",
		"options": [
			{
				"access": 2,
				"description": "Set to false to disable the legacy integration (highly recommended), will change structure of the published payload (default true).",
				"name": "legacy",
				"property": "legacy",
				"type": "binary",
				"value_off": false,
				"value_on": true
			},
			{
				"description": "Simulate a brightness value. If this device provides a brightness_move_up or brightness_move_down action it is possible to specify the update interval and delta.Only works when legacy is false.",
				"features": [
					{
						"access": 2,
						"description": "Delta per interval, 20 by default",
						"name": "delta",
						"property": "delta",
						"type": "numeric",
						"value_min": 0
					},
					{
						"access": 2,
						"description": "Interval duration",
						"name": "interval",
						"property": "interval",
						"type": "numeric",
						"unit": "ms",
						"value_min": 0
					}
				],
				"name": "simulated_brightness",
				"property": "simulated_brightness",
				"type": "composite"
			}
		],
		"supports_ota": true,
		"vendor": "Philips"
	},
	"endpoints": {
		"1": {
			"bindings": [
				{
					"cluster": "genOnOff",
					"target": {
						"endpoint": 1,
						"ieee_address": "0xdeadbeef",
						"type": "endpoint"
					}
				},
				{
					"cluster": "genLevelCtrl",
					"target": {
						"endpoint": 1,
						"ieee_address": "0xdeadbeef",
						"type": "endpoint"
					}
				}
			],
			"clusters": {
				"input": [
					"genBasic"
				],
				"output": [
					"genBasic",
					"genIdentify",
					"genGroups",
					"genOnOff",
					"genLevelCtrl",
					"genScenes"
				]
			},
			"configured_reportings": [],
			"scenes": []
		},
		"2": {
			"bindings": [
				{
					"cluster": "manuSpecificUbisysDeviceSetup",
					"target": {
						"endpoint": 1,
						"ieee_address": "0xdeadbeef",
						"type": "endpoint"
					}
				},
				{
					"cluster": "genPowerCfg",
					"target": {
						"endpoint": 1,
						"ieee_address": "0xdeadbeef",
						"type": "endpoint"
					}
				}
			],
			"clusters": {
				"input": [
					"genBasic",
					"genPowerCfg",
					"genIdentify",
					"genBinaryInput",
					"manuSpecificPhilips"
				],
				"output": [
					"genOta"
				]
			},
			"configured_reportings": [
				{
					"attribute": "batteryPercentageRemaining",
					"cluster": "genPowerCfg",
					"maximum_report_interval": 62000,
					"minimum_report_interval": 3600,
					"reportable_change": 0
				}
			],
			"scenes": []
		}
	},
	"friendly_name": "Living Room Dimmer Switch",
	"ieee_address": "0xdeadbeef",
	"interview_completed": true,
	"interviewing": false,
	"manufacturer": "Philips",
	"model_id": "RWL021",
	"network_address": 48525,
	"power_source": "Battery",
	"software_build_id": "6.1.1.28573",
	"supported": true,
	"type": "EndDevice"
}`)
