package zigbee

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/wh/v1/bridges"
	"github.com/jimjibone/wh/v1/bridges/attributes"
	"github.com/jimjibone/wh/v1/bridges/services"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
)

// PermitJoinSeconds is how long zigbee2mqtt keeps the network open for new
// devices when pairing is enabled. 254 seconds is the maximum allowed by the
// Zigbee spec and matches the zigbee2mqtt frontend default.
const PermitJoinSeconds = 254

// BridgeInfo is the interesting subset of the zigbee2mqtt bridge/info payload.
type BridgeInfo struct {
	Version     string `json:"version"`
	Coordinator struct {
		IEEEAddress string `json:"ieee_address"`
		Type        string `json:"type"`
	} `json:"coordinator"`
	PermitJoin bool `json:"permit_join"`
	// PermitJoinEnd is the epoch time in milliseconds at which permit join
	// will be disabled again (zigbee2mqtt 2.x).
	PermitJoinEnd *float64 `json:"permit_join_end"`
	// PermitJoinTimeout is the number of seconds remaining until permit join
	// is disabled again (zigbee2mqtt 1.x).
	PermitJoinTimeout *float64 `json:"permit_join_timeout"`
}

// BridgeDevice represents the zigbee2mqtt bridge itself as a woodhouse device.
// It exposes a pairing service that opens the zigbee network for new devices
// to join and reports how long the network will remain open.
type BridgeDevice struct {
	log        *log.Context
	bridge     *bridges.Bridge
	added      bool
	baseUrl    string
	requests   func(ZigbeeRequest)
	correlator *ResponseCorrelator

	dev              *bridges.Device
	pairing          *services.Generic
	pairingEnabled   *attributes.Bool
	pairingRemaining *attributes.Duration
	pairingEnds      *attributes.Time

	mu       sync.Mutex
	deadline time.Time
}

func NewBridgeDevice(info BridgeInfo, client *bridges.Bridge, baseUrl string, requests func(ZigbeeRequest), correlator *ResponseCorrelator) *BridgeDevice {
	id := info.Coordinator.IEEEAddress
	if id == "" {
		id = "zigbee2mqtt-bridge"
	}

	dev := &BridgeDevice{
		log:              log.NewContext(log.DefaultLogger, id, log.DebugLevel),
		bridge:           client,
		baseUrl:          baseUrl,
		requests:         requests,
		correlator:       correlator,
		dev:              bridges.NewDevice(id, clientsapi.Device_BRIDGE),
		pairing:          services.NewGeneric("pairing"),
		pairingEnabled:   attributes.NewBool("enabled", clientsapi.Permissions_PERM_READWRITE, attributes.Required),
		pairingRemaining: attributes.NewDuration("remaining", clientsapi.Permissions_PERM_READONLY, attributes.Optional, 0, PermitJoinSeconds*time.Second, time.Second),
		pairingEnds:      attributes.NewTime("ends", clientsapi.Permissions_PERM_READONLY, attributes.Optional),
	}

	dev.pairing.AddAttribute(
		dev.pairingEnabled,
		dev.pairingRemaining,
		dev.pairingEnds,
	)
	dev.pairing.OnAction(dev.handleAction)
	dev.dev.AddService(dev.pairing)
	dev.pairingEnabled.Set(false)

	dev.log.Infof("created bridge device")

	go dev.countdown()

	dev.UpdateInfo(info)

	return dev
}

func (dev *BridgeDevice) Device() *bridges.Device { return dev.dev }

func (dev *BridgeDevice) UpdateInfo(info BridgeInfo) {
	dev.dev.Info.Name.Set("Zigbee Network")
	dev.dev.Info.Model.Set(info.Coordinator.Type)
	dev.dev.Info.SerialNumber.Set(info.Coordinator.IEEEAddress)
	dev.dev.Info.FirmwareVersion.Set(info.Version)
	dev.dev.Info.WebUrl.Set(fmt.Sprintf("http://%s/", dev.baseUrl))

	dev.log.Debugf("bridge info: %+v", info)

	now := time.Now()
	var deadline time.Time
	if info.PermitJoin {
		if info.PermitJoinEnd != nil {
			deadline = time.UnixMilli(int64(*info.PermitJoinEnd))
		} else if info.PermitJoinTimeout != nil {
			deadline = now.Add(time.Duration(*info.PermitJoinTimeout * float64(time.Second)))
		}
	}

	dev.mu.Lock()
	dev.deadline = deadline
	dev.mu.Unlock()

	dev.pairingEnabled.Set(info.PermitJoin)
	dev.pairingEnds.Set(deadline)
	dev.pairingRemaining.Set(remainingUntil(deadline, now))

	// Add this device to the client if not done already.
	if !dev.added {
		dev.added = true
		err := dev.bridge.AddDevice(dev.dev)
		if err != nil {
			dev.log.Fatalf("failed to add device: %s", err)
		}
	}
}

// countdown keeps the remaining attribute ticking down while pairing is
// enabled. zigbee2mqtt 2.x only reports the pairing deadline when it changes,
// so the countdown itself must be maintained here. The enabled attribute is
// left alone; zigbee2mqtt republishes bridge/info once permit join actually
// ends and UpdateInfo picks the change up from there.
func (dev *BridgeDevice) countdown() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		dev.mu.Lock()
		deadline := dev.deadline
		dev.mu.Unlock()
		if deadline.IsZero() {
			continue
		}
		dev.pairingRemaining.Set(remainingUntil(deadline, time.Now()))
	}
}

func remainingUntil(deadline, now time.Time) time.Duration {
	if deadline.IsZero() {
		return 0
	}
	remaining := deadline.Sub(now)
	if remaining < 0 {
		return 0
	}
	return remaining.Round(time.Second)
}

func (dev *BridgeDevice) handleAction(request *clientsapi.ActionRequest, feedback func(*clientsapi.ActionResponse)) error {
	for _, val := range request.GetValues() {
		if val.GetId() != dev.pairingEnabled.ID() {
			// Every other pairing attribute is read-only.
			return services.ErrReadOnly
		}
		if val.GetBool() == nil {
			return services.ErrIncorrectTypeFor(dev.pairingEnabled)
		}
		if err := dev.handlePermitJoin(val.GetBool().GetValue()); err != nil {
			return err
		}
	}
	return nil
}

// handlePermitJoin forwards a permit join request to zigbee2mqtt and waits
// for its response. The pairing attributes are not updated here; zigbee2mqtt
// republishes bridge/info when permit join changes and UpdateInfo picks up
// the new state from there.
func (dev *BridgeDevice) handlePermitJoin(enable bool) error {
	if dev.correlator == nil {
		return fmt.Errorf("pairing not supported")
	}

	tx, ch := dev.correlator.NewTransaction()
	defer dev.correlator.Forget(tx)

	seconds := 0
	if enable {
		seconds = PermitJoinSeconds
	}

	// The value field drives zigbee2mqtt 1.x while the time field drives
	// 2.x; each version ignores the field it does not use.
	payload, err := json.Marshal(struct {
		Value       bool   `json:"value"`
		Time        int    `json:"time"`
		Transaction string `json:"transaction"`
	}{
		Value:       enable,
		Time:        seconds,
		Transaction: tx,
	})
	if err != nil {
		return err
	}

	dev.log.Infof("setting permit join to %t for %ds (transaction %s)", enable, seconds, tx)
	dev.requests(ZigbeeRequest{
		Topic:   "bridge/request/permit_join",
		Payload: payload,
	})

	select {
	case res := <-ch:
		if !res.OK {
			return fmt.Errorf("zigbee2mqtt permit join failed: %s", res.Error)
		}
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timed out waiting for zigbee2mqtt permit join response")
	}
}
