package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-shelly/shelly_v1"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-shelly/shelly_v2"
	"github.com/jimjibone/woodhouse-4/wh"
)

func shellyStuff(ctx context.Context, bridge *wh.Bridge, doUpdatesChan <-chan bool) error {
	log.Printf("shelly started")
	defer log.Printf("shelly finished")

	// Use mDNS to discover Shelly devices.
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)

	err = resolver.Browse(ctx, "_http._tcp", ".local", entries)
	if err != nil {
		return fmt.Errorf("failed to browse: %w", err)
	}

	var devices_v1 []shelly_v1.Device
	var devices_v2 []shelly_v2.Device

	// Get devices to close when we're exiting.
	defer func() {
		for _, device := range devices_v2 {
			device.Close()
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	doUpdates := false

	for {
		select {
		case <-ctx.Done():
			return nil

		case doUpdates = <-doUpdatesChan:

		case entry := <-entries:
			deviceID, device_v1, device_v2 := handleDiscovery(bridge, entry)
			if device_v1 != nil {
				devices_v1 = append(devices_v1, device_v1)
				bridge.AddDevice(deviceID, device_v1)
			}
			if device_v2 != nil {
				devices_v2 = append(devices_v2, device_v2)
				bridge.AddDevice(deviceID, device_v2)
			}

		case <-ticker.C:
			// Only do updates when connected to woodhouse.
			if doUpdates {
				for _, device := range devices_v1 {
					device.UpdateState(false)
				}
			}
		}
	}
}

func handleDiscovery(bridge *wh.Bridge, entry *zeroconf.ServiceEntry) (deviceID string, device_v1 shelly_v1.Device, device_v2 shelly_v2.Device) {
	ipstring := ""
	if len(entry.AddrIPv4) > 0 {
		ipstring = entry.AddrIPv4[0].String()
	}
	hostname := strings.TrimSuffix(entry.HostName, ".local.")

	// Gen 1 Device API - https://shelly-api-docs.shelly.cloud/gen1
	if strings.Contains(hostname, "shelly") {
		rest := shelly_v1.Rest{IP: ipstring}
		settings, err := rest.GetSettings()
		if err != nil {
			log.Printf("ERROR: failed to get settings for %s: %s", ipstring, err)
			return "", nil, nil
		}

		log.Printf("discovered v1 %s at http://%s (name: %s)", hostname, ipstring, settings.Name)

		return hostname, shelly_v1.Generate(settings.Device.Type, hostname, ipstring), nil
	}

	// Gen 2 Device API - https://shelly-api-docs.shelly.cloud/gen2
	if strings.Contains(hostname, "ShellyPlus") {
		rest := shelly_v2.Rest{IP: ipstring}
		info, err := rest.GetShelly()
		if err != nil {
			log.Printf("ERROR: failed to get shelly info for %s: %s", ipstring, err)
			return "", nil, nil
		}

		log.Printf("discovered v2 %s at http://%s (name: %s, app: %s)", hostname, ipstring, info.Name, info.App)

		return hostname, nil, shelly_v2.NewShellyPlusDevice(hostname, ipstring, info.Name, info.App)
	}

	log.Printf("other thing %s at http://%s (txt: %s)", hostname, ipstring, entry.Text)

	return "", nil, nil
}
