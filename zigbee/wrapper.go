package zigbee

type Wrapper interface {
	UpdateInfo(info DeviceInfo) (handled []HandledExpose)
	UpdateState(info DeviceState) (handled []string)
}

type HandledExpose struct {
	Type     string
	Property string
}
