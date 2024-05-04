package shelly_v2

import (
	"fmt"
	"time"
)

type InputValue struct {
	ID        int
	Name      string
	Type      string
	Invert    bool
	Timestamp time.Time
	State     bool
}

func (v *InputValue) GetName() string {
	if v.Name == "" {
		return fmt.Sprintf("Input %d", v.ID)
	}
	return v.Name
}

type ScriptValue struct {
	ID        int
	Name      string
	Timestamp time.Time
	Enable    bool
	Running   bool
}

func (v *ScriptValue) GetName() string {
	if v.Name == "" {
		return fmt.Sprintf("Script %d", v.ID)
	}
	return v.Name
}

type SwitchValue struct {
	ID            int
	Name          string
	Timestamp     time.Time
	State         bool
	AveragePower  float64
	Voltage       float64
	Current       float64
	AverageEnergy struct {
		Total           float64
		ByMinute        []float64
		MinuteTimestamp int
	}
	Temperature float64 // Centigrade
}

func (v *SwitchValue) GetName() string {
	if v.Name == "" {
		return fmt.Sprintf("Switch %d", v.ID)
	}
	return v.Name
}
