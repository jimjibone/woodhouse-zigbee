package shelly_v2

import (
	"encoding/json"
	"fmt"
	"strings"
)

type NotificationFrame struct {
	Src          string                     `json:"src"`
	Dst          string                     `json:"dst"`
	Method       string                     `json:"method"`
	Params       map[string]json.RawMessage `json:"params"`
	NotifyStatus *NotifyStatus              `json:"-"`
}

type NotifyStatus struct {
	Timestamp float64
	Inputs    []GetStatusResponseInput
	Scripts   []GetStatusResponseScript
	Switches  []GetStatusResponseSwitch
}

func (m *NotificationFrame) UnmarshalJSON(data []byte) error {
	type Tmp NotificationFrame
	tmp := Tmp{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	switch tmp.Method {
	case "NotifyStatus":
		tmp.NotifyStatus = &NotifyStatus{}

		for field, data := range tmp.Params {
			if strings.Contains(field, "ts") {
				var msg float64
				err = json.Unmarshal(data, &msg)
				if err != nil {
					return err
				}
				tmp.NotifyStatus.Timestamp = msg
			} else if strings.Contains(field, "input") {
				msg := GetStatusResponseInput{}
				err = json.Unmarshal(data, &msg)
				if err != nil {
					return err
				}
				tmp.NotifyStatus.Inputs = append(tmp.NotifyStatus.Inputs, msg)
			} else if strings.Contains(field, "switch") {
				msg := GetStatusResponseSwitch{}
				err = json.Unmarshal(data, &msg)
				if err != nil {
					return err
				}
				tmp.NotifyStatus.Switches = append(tmp.NotifyStatus.Switches, msg)
			}
		}

	default:
		return fmt.Errorf("unknown notification method %q", tmp.Method)
	}

	*m = NotificationFrame(tmp)

	return nil
}
