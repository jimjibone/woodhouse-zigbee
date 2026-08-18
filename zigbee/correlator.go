package zigbee

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ResponseResult is the outcome of a zigbee2mqtt bridge request.
type ResponseResult struct {
	OK    bool
	Error string
}

// ResponseCorrelator matches zigbee2mqtt bridge responses to their pending
// requests using the transaction field. Create one per transport and share it
// with the devices so they can wait for the outcome of bridge requests.
type ResponseCorrelator struct {
	mu      sync.Mutex
	seq     uint64
	pending map[string]chan ResponseResult
}

func NewResponseCorrelator() *ResponseCorrelator {
	return &ResponseCorrelator{
		pending: make(map[string]chan ResponseResult),
	}
}

// NewTransaction registers a new pending transaction, returning its ID (to be
// included in the request payload) and the channel that will receive the
// result. Callers should defer Forget with the returned ID.
func (c *ResponseCorrelator) NewTransaction() (string, <-chan ResponseResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	id := fmt.Sprintf("wh-%d", c.seq)
	ch := make(chan ResponseResult, 1)
	c.pending[id] = ch
	return id, ch
}

// Forget removes a pending transaction, e.g. after a timeout.
func (c *ResponseCorrelator) Forget(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pending, id)
}

// HandleResponse parses a zigbee2mqtt bridge response payload and resolves
// the matching pending transaction, if any. Responses without a known
// transaction are ignored; they belong to other zigbee2mqtt clients.
func (c *ResponseCorrelator) HandleResponse(payload []byte) {
	response := struct {
		Status      string `json:"status"`
		Error       string `json:"error"`
		Transaction string `json:"transaction"`
	}{}
	if err := json.Unmarshal(payload, &response); err != nil {
		return
	}
	if response.Transaction == "" {
		return
	}

	c.mu.Lock()
	ch, found := c.pending[response.Transaction]
	if found {
		delete(c.pending, response.Transaction)
	}
	c.mu.Unlock()
	if !found {
		return
	}

	result := ResponseResult{OK: response.Status == "ok", Error: response.Error}
	if !result.OK && result.Error == "" {
		result.Error = "unknown error"
	}
	ch <- result
}
