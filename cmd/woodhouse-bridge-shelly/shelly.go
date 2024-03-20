package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-shelly/shelly_v1"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-bridge-shelly/shelly_v2"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh"
)

func shellyStuff(ctx context.Context, bridge *wh.Bridge, doUpdatesChan <-chan bool) error {
	log.Infof("shelly started")
	defer log.Infof("shelly finished")

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

	devices_v1 := make(map[string]shelly_v1.Device)
	devices_v2 := make(map[string]shelly_v2.Device)

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
			ip := ""
			if len(entry.AddrIPv4) > 0 {
				ip = entry.AddrIPv4[0].String()
			}
			hostname := strings.TrimSuffix(entry.HostName, ".local.")

			exists := false
			if _, found := devices_v1[hostname]; found {
				exists = true
			}
			if _, found := devices_v2[hostname]; found {
				exists = true
			}

			if !exists {
				deviceID, device_v1, device_v2 := handleDiscovery(bridge, ip, hostname)
				if device_v1 != nil {
					devices_v1[deviceID] = device_v1
					bridge.AddDevice(deviceID, device_v1)
				}
				if device_v2 != nil {
					devices_v2[deviceID] = device_v2
					bridge.AddDevice(deviceID, device_v2)
				}
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

func handleDiscovery(bridge *wh.Bridge, ip, hostname string) (deviceID string, device_v1 shelly_v1.Device, device_v2 shelly_v2.Device) {
	// Gen 1 Device API - https://shelly-api-docs.shelly.cloud/gen1
	if strings.Contains(hostname, "shelly") {
		rest := shelly_v1.Rest{IP: ip}
		settings, err := rest.GetSettings()
		if err != nil {
			log.Errorf("failed to get settings for %s: %s", ip, err)
			return "", nil, nil
		}

		log.Infof("discovered v1 %s at http://%s (name: %s)", hostname, ip, settings.Name)

		return hostname, shelly_v1.Generate(settings.Device.Type, hostname, ip), nil
	}

	// Gen 2 Device API - https://shelly-api-docs.shelly.cloud/gen2
	if strings.Contains(hostname, "ShellyPlus") {
		rest := shelly_v2.Rest{IP: ip}
		info, err := rest.GetShelly()
		if err != nil {
			log.Errorf("failed to get shelly info for %s: %s", ip, err)
			return "", nil, nil
		}

		log.Infof("discovered v2 %s at http://%s (name: %s, app: %s)", hostname, ip, info.Name, info.App)

		return hostname, nil, shelly_v2.NewShellyPlusDevice(hostname, ip, info.Name, info.App)
	}

	log.Warnf("other thing %s at http://%s", hostname, ip)

	return "", nil, nil
}
