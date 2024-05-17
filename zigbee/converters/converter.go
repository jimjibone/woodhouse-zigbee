package converters

import (
	api "github.com/jimjibone/woodhouse-4/api/go"
)

type Converter interface {
	String() string
	Unmarshal(value []byte) (*api.DeviceValue, error)
	Marshal(value *api.DeviceValue) ([]byte, error)
}
