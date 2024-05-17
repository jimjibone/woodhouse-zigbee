package zigbee_test

import (
	"encoding/json"
	"testing"

	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-zigbee/zigbee"
)

func TestDeviceInfo(t *testing.T) {
	// Load device info.
	var info zigbee.DeviceInfo
	err := json.Unmarshal(DeviceInfoData1, &info)
	if err != nil {
		t.Fatalf("failed to unmarshal device info: %v", err)
	}
	t.Logf("device info: %+v", info)

	dev := zigbee.NewZigbeeDevice("", nil)
	err = dev.UpdateInfo(info)
	if err != nil {
		t.Fatalf("failed to update device info: %v", err)
	}
	t.Logf("device: %+v", dev)

	// Load device state 1.
	var state zigbee.DeviceState
	err = json.Unmarshal(DeviceStateData1, &state)
	if err != nil {
		t.Fatalf("failed to unmarshal device state: %v", err)
	}
	t.Logf("device state 1: %+v", state)

	err = dev.UpdateState(state)
	if err != nil {
		t.Fatalf("failed to update device info: %v", err)
	}
	t.Logf("device 1: %+v", dev)

	// Load device state 2.
	err = json.Unmarshal(DeviceStateData2, &state)
	if err != nil {
		t.Fatalf("failed to unmarshal device state: %v", err)
	}
	t.Logf("device state 2: %+v", state)

	err = dev.UpdateState(state)
	if err != nil {
		t.Fatalf("failed to update device info: %v", err)
	}
	t.Logf("device 2: %+v", dev)
}

var DeviceStateData1 = []byte(`{
	"battery": 36,
	"humidity": 85.87,
	"last_seen": "2022-11-16T19:41:20.490Z",
	"linkquality": 111,
	"power_outage_count": 1082,
	"enabled": true,
	"pressure": 974.6,
	"temperature": 10.63,
	"voltage": 2975,
	"contact": true,
	"state": "ON",
	"effect": "blink",
	"brightness": 254,
	"color": {
		"h": 32,
		"hue": 32,
		"s": 82,
		"saturation": 82,
		"x": 0.4599,
		"y": 0.4106
	},
	"color_mode": "color_temp",
	"color_temp": 370,
	"ADC": "1426,1421,1,196,-32768,2000,0,500,100,207,209,213",
	"ALG": "5ef6,1,200,196,10,0,500,4413,79984,100,100",
	"boost": "Down",
	"keypad_lockout": "unlock",
	"local_temperature": 19.6,
	"occupied_heating_setpoint": 20,
	"pi_heating_demand": 100
}`)

var DeviceStateData2 = []byte(`{
	"battery": 35,
	"humidity": 84.57,
	"last_seen": "2022-11-16T19:42:20.490Z",
	"linkquality": 110,
	"power_outage_count": 1083,
	"enabled": false,
	"pressure": 974.0,
	"temperature": 10.33,
	"voltage": 2974,
	"contact": false,
	"state": "OFF",
	"effect": "okay",
	"brightness": 244,
	"color": {
		"h": 31,
		"hue": 31,
		"s": 81,
		"saturation": 81,
		"x": 0.4499,
		"y": 0.4006
	},
	"color_mode": "color_temp",
	"color_temp": 360,
	"ADC": "1426,1421,1,196,-32768,2000,0,500,100,207,209,213",
	"ALG": "5ef7,1,200,196,10,0,500,4413,79984,100,100",
	"boost": "Up",
	"keypad_lockout": "unlock",
	"local_temperature": 20.6,
	"occupied_heating_setpoint": 21,
	"pi_heating_demand": 90
}`)

var DeviceInfoData1 = []byte(`{
	"date_code": "20191205",
	"definition": {
		"description": "Aqara temperature, humidity and pressure sensor",
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
				"description": "Measured temperature value",
				"name": "temperature",
				"property": "temperature",
				"type": "numeric",
				"unit": "°C"
			},
			{
				"access": 1,
				"description": "Measured relative humidity",
				"name": "humidity",
				"property": "humidity",
				"type": "numeric",
				"unit": "%"
			},
			{
				"access": 1,
				"description": "The measured atmospheric pressure",
				"name": "pressure",
				"property": "pressure",
				"type": "numeric",
				"unit": "hPa"
			},
			{
				"access": 1,
				"description": "Voltage of the battery in millivolts",
				"name": "voltage",
				"property": "voltage",
				"type": "numeric",
				"unit": "mV"
			},
			{
				"access": 1,
				"description": "Indicates if the contact is closed (= true) or open (= false)",
				"name": "contact",
				"property": "contact",
				"type": "binary",
				"value_off": true,
				"value_on": false
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
			},
			{
				"access": 2,
				"description": "Triggers an effect on the light (e.g. make light blink for a few seconds)",
				"name": "effect",
				"property": "effect",
				"type": "enum",
				"values": [
					"blink",
					"breathe",
					"okay",
					"channel_change",
					"finish_effect",
					"stop_effect"
				]
			},
			{
				"type": "light",
				"features": [
					{
						"access": 7,
						"description": "On/off state of this light",
						"name": "state",
						"property": "state",
						"type": "binary",
						"value_off": "OFF",
						"value_on": "ON",
						"value_toggle": "TOGGLE"
					},
					{
						"access": 7,
						"description": "Brightness of this light",
						"name": "brightness",
						"property": "brightness",
						"type": "numeric",
						"value_max": 254,
						"value_min": 0
					},
					{
						"access": 7,
						"description": "Color temperature of this light",
						"name": "color_temp",
						"presets": [
							{
								"description": "Coolest temperature supported",
								"name": "coolest",
								"value": 153
							},
							{
								"description": "Cool temperature (250 mireds / 4000 Kelvin)",
								"name": "cool",
								"value": 250
							},
							{
								"description": "Neutral temperature (370 mireds / 2700 Kelvin)",
								"name": "neutral",
								"value": 370
							},
							{
								"description": "Warm temperature (454 mireds / 2200 Kelvin)",
								"name": "warm",
								"value": 454
							},
							{
								"description": "Warmest temperature supported",
								"name": "warmest",
								"value": 555
							}
						],
						"property": "color_temp",
						"type": "numeric",
						"unit": "mired",
						"value_max": 555,
						"value_min": 153
					},
					{
						"access": 7,
						"description": "Color temperature after cold power on of this light",
						"name": "color_temp_startup",
						"presets": [
							{
								"description": "Coolest temperature supported",
								"name": "coolest",
								"value": 153
							},
							{
								"description": "Cool temperature (250 mireds / 4000 Kelvin)",
								"name": "cool",
								"value": 250
							},
							{
								"description": "Neutral temperature (370 mireds / 2700 Kelvin)",
								"name": "neutral",
								"value": 370
							},
							{
								"description": "Warm temperature (454 mireds / 2200 Kelvin)",
								"name": "warm",
								"value": 454
							},
							{
								"description": "Warmest temperature supported",
								"name": "warmest",
								"value": 555
							},
							{
								"description": "Restore previous color_temp on cold power on",
								"name": "previous",
								"value": 65535
							}
						],
						"property": "color_temp_startup",
						"type": "numeric",
						"unit": "mired",
						"value_max": 555,
						"value_min": 153
					},
					{
						"description": "Color of this light in the CIE 1931 color space (x/y)",
						"features": [
							{
								"access": 7,
								"name": "x",
								"property": "x",
								"type": "numeric"
							},
							{
								"access": 7,
								"name": "y",
								"property": "y",
								"type": "numeric"
							}
						],
						"name": "color_xy",
						"property": "color",
						"type": "composite"
					},
					{
						"description": "Color of this light expressed as hue/saturation",
						"features": [
							{
								"access": 7,
								"name": "hue",
								"property": "hue",
								"type": "numeric"
							},
							{
								"access": 7,
								"name": "saturation",
								"property": "saturation",
								"type": "numeric"
							}
						],
						"name": "color_hs",
						"property": "color",
						"type": "composite"
					}
				]
			},
			{
				"features": [
					{
						"access": 7,
						"description": "Temperature setpoint",
						"name": "occupied_heating_setpoint",
						"property": "occupied_heating_setpoint",
						"type": "numeric",
						"unit": "°C",
						"value_max": 30,
						"value_min": 7,
						"value_step": 1
					},
					{
						"access": 1,
						"description": "Current temperature measured on the device",
						"name": "local_temperature",
						"property": "local_temperature",
						"type": "numeric",
						"unit": "°C"
					},
					{
						"access": 1,
						"description": "Mode of this device",
						"name": "system_mode",
						"property": "system_mode",
						"type": "enum",
						"values": [
							"off",
							"auto",
							"heat"
						]
					},
					{
						"access": 1,
						"description": "The current running state",
						"name": "running_state",
						"property": "running_state",
						"type": "enum",
						"values": [
							"idle",
							"heat"
						]
					},
					{
						"access": 1,
						"description": "Position of the valve (= demanded heat) where 0% is fully closed and 100% is fully open",
						"name": "pi_heating_demand",
						"property": "pi_heating_demand",
						"type": "numeric",
						"unit": "%",
						"value_max": 100,
						"value_min": 0
					}
				],
				"type": "climate"
			}
		],
		"model": "WSDCGQ11LM",
		"options": [
			{
				"access": 2,
				"description": "Number of digits after decimal point for temperature, takes into effect on next report of device.",
				"name": "temperature_precision",
				"property": "temperature_precision",
				"type": "numeric",
				"value_max": 3,
				"value_min": 0
			},
			{
				"access": 2,
				"description": "Calibrates the temperature value (absolute offset), takes into effect on next report of device.",
				"name": "temperature_calibration",
				"property": "temperature_calibration",
				"type": "numeric"
			},
			{
				"access": 2,
				"description": "Number of digits after decimal point for humidity, takes into effect on next report of device.",
				"name": "humidity_precision",
				"property": "humidity_precision",
				"type": "numeric",
				"value_max": 3,
				"value_min": 0
			},
			{
				"access": 2,
				"description": "Calibrates the humidity value (absolute offset), takes into effect on next report of device.",
				"name": "humidity_calibration",
				"property": "humidity_calibration",
				"type": "numeric"
			},
			{
				"access": 2,
				"description": "Number of digits after decimal point for pressure, takes into effect on next report of device.",
				"name": "pressure_precision",
				"property": "pressure_precision",
				"type": "numeric",
				"value_max": 3,
				"value_min": 0
			},
			{
				"access": 2,
				"description": "Calibrates the pressure value (absolute offset), takes into effect on next report of device.",
				"name": "pressure_calibration",
				"property": "pressure_calibration",
				"type": "numeric"
			}
		],
		"supports_ota": false,
		"vendor": "Xiaomi"
	},
	"endpoints": {
		"1": {
			"bindings": [],
			"clusters": {
				"input": [
					"genBasic",
					"genIdentify",
					"65535",
					"msTemperatureMeasurement",
					"msPressureMeasurement",
					"msRelativeHumidity"
				],
				"output": [
					"genBasic",
					"genGroups",
					"65535"
				]
			},
			"configured_reportings": [],
			"scenes": []
		}
	},
	"friendly_name": "Environment Sensor",
	"ieee_address": "0xdeadbeef",
	"interview_completed": true,
	"interviewing": false,
	"manufacturer": "LUMI",
	"model_id": "lumi.weather",
	"network_address": 12345,
	"power_source": "Battery",
	"software_build_id": "3000-0001",
	"supported": true,
	"type": "EndDevice"
}`)
