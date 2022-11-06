package shelly_v2

type SwitchSet struct {
	ID          int      `json:"id"`           // ID of the Switch component instance. Required.
	On          bool     `json:"on"`           // True for switch on, false otherwise. Required.
	ToggleAfter *float64 `json:"toggle_after"` // Optional flip-back timer in seconds. Optional.
}
