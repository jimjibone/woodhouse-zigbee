package shelly_v2

import "github.com/jimjibone/woodhouse-4/wh"

type Device interface {
	wh.Device
	Close()
}
