package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-shelly/shelly"
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

	var devices []shelly.Device

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	doUpdates := false

	for {
		select {
		case <-ctx.Done():
			return nil

		case doUpdates = <-doUpdatesChan:

		case entry := <-entries:
			deviceID, device := handleDiscovery(bridge, entry)
			if device != nil {
				devices = append(devices, device)
				bridge.AddDevice(deviceID, device)
			}

		case <-ticker.C:
			// Only do updates when connected to woodhouse.
			if doUpdates {
				for _, device := range devices {
					device.UpdateState(false)
				}
			}
		}
	}
}

func handleDiscovery(bridge *wh.Bridge, entry *zeroconf.ServiceEntry) (deviceID string, device shelly.Device) {
	ipstring := ""
	if len(entry.AddrIPv4) > 0 {
		ipstring = entry.AddrIPv4[0].String()
	}
	hostname := strings.TrimSuffix(entry.HostName, ".local.")

	if strings.Contains(hostname, "shelly") {
		rest := shelly.Rest{IP: ipstring}
		settings, err := rest.GetSettings()
		if err != nil {
			log.Printf("ERROR: failed to get settings for %s: %s", ipstring, err)
			return "", nil
		}

		log.Printf("discovered %s at http://%s (name: %s)", hostname, ipstring, settings.Name)

		return hostname, shelly.Generate(settings.Device.Type, hostname, ipstring)
	}
	return "", nil
}
