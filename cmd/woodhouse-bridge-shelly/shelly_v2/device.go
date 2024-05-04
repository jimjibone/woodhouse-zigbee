package shelly_v2

import (
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1"
)

type Device interface {
	ID() string
	Close()
}

func Generate(app, hostname, ip, name string, client *wh.Client) Device {
	switch app {
	case "Plus1PM":
		return NewShellyPlus1PM(hostname, ip, client)
	case "Plus2PM":
		return NewShellyPlus2PM(hostname, ip, client)
	// case "PlusPlugUK":
	// 	return NewShellyPlusPlugUK(hostname, ip, name, client)
	default:
		log.Warnf("unknown app %q for %s (%s)", app, hostname, name)
	}
	return nil
}
