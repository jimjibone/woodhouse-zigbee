package zigbee_old_test

import (
	"encoding/json"
	"testing"

	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-zigbee/zigbee_old"
)

func TestDeviceInfo1(t *testing.T) {
	var info zigbee_old.DeviceInfo
	if err := json.Unmarshal(DeviceInfoData1, &info); err != nil {
		t.Fatalf("failed to unmarshal device info: %v", err)
	}
	t.Logf("device info: %s", info.LongString("  "))
}

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
				"description": "Link quality (signal strength)",
				"name": "linkquality",
				"property": "linkquality",
				"type": "numeric",
				"unit": "lqi",
				"value_max": 255,
				"value_min": 0
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
	"network_address": 53000,
	"power_source": "Battery",
	"software_build_id": "3000-0001",
	"supported": true,
	"type": "EndDevice"
}`)
