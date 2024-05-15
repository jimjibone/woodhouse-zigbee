package shelly_v2

import (
	"encoding/json"
	"fmt"

	"github.com/jimjibone/woodhouse-4/log"
)

type FrameType int

const (
	UnknownFrameType FrameType = iota
	ResponseFrameType
	NotificationFrameType
)

func DetectFrameType(data []byte) FrameType {
	tmp := struct {
		ID     *FrameID `json:"id"`
		Method *string  `json:"method"`
	}{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		log.Errorf("DetectFrameType unmarshal failed: %s", err)
		return UnknownFrameType
	}
	if tmp.ID != nil {
		return ResponseFrameType
	}
	if tmp.Method != nil {
		return NotificationFrameType
	}
	return UnknownFrameType
}

type RequestFrame struct {
	JsonRpc string      `json:"jsonrpc"`          // 2.0. The version of jsonrpc used. May be omitted.
	ID      FrameID     `json:"id"`               // Identifier of this request, will be used to match the response frame. Required.
	Src     string      `json:"src"`              // Name of the source of the request (you can choose whatever string you like to identify you as the source of the request). Required.
	Method  string      `json:"method"`           // Name of the procedure to be called. Required.
	Params  interface{} `json:"params,omitempty"` // Parameters that the method takes (if any). Optional.
}

type ResponseFrame struct {
	ID     FrameID         `json:"id"`
	Src    string          `json:"src"`
	Dst    string          `json:"dst"`
	Result json.RawMessage `json:"result"`
	Error  struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type FrameID string

func (id *FrameID) UnmarshalJSON(p []byte) error {
	var tmp any
	err := json.Unmarshal(p, &tmp)
	if err != nil {
		return err
	}
	switch tmp := tmp.(type) {
	case string:
		*id = FrameID(tmp)
	case float64:
		*id = FrameID(fmt.Sprintf("%.0f", tmp))
	default:
		log.Errorf("unknown type %T for %+v for %q", tmp, tmp, p)
	}
	return nil
}
