package shelly_v2

import (
	"encoding/json"
	"log"
)

type FrameType int

const (
	UnknownFrameType FrameType = iota
	ResponseFrameType
	NotificationFrameType
)

func DetectFrameType(data []byte) FrameType {
	tmp := struct {
		ID     *int    `json:"id"`
		Method *string `json:"method"`
	}{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		log.Println("ERROR: IsResponseFrame unmarshal:", err)
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
	JsonRpc string      `json:"jsonrpc"` // 2.0. The version of jsonrpc used. May be omitted.
	ID      int         `json:"id"`      // Identifier of this request, will be used to match the response frame. Required.
	Src     string      `json:"src"`     // Name of the source of the request (you can choose whatever string you like to identify you as the source of the request). Required.
	Method  string      `json:"method"`  // Name of the procedure to be called. Required.
	Params  interface{} `json:"params"`  // Parameters that the method takes (if any). Optional.
}

type ResponseFrame struct {
	ID     int             `json:"id"`
	Src    string          `json:"src"`
	Dst    string          `json:"dst"`
	Result json.RawMessage `json:"result"`
	Error  struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
