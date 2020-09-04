package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"
	"k8s.io/klog/v2"
)

var requestID int64 = 0

func nextResponseID() int {
	return int(atomic.AddInt64(&requestID, 1))
}

type response struct {
	rpc.Response
	rpc.Body
}

type result struct {
	Return interface{} `json:"_ret_"`
}

func raritanResultJSON(w http.ResponseWriter, r interface{}) {
	jsonResult(w, result{
		Return: r,
	})
}

func jsonError(w http.ResponseWriter, e rpc.Error) {
	jsonResponse(w, rpc.Response{
		Error: &e,
	})
}

func jsonMethodNotFound(w http.ResponseWriter, method string) {
	jsonError(w, rpc.Error{
		Code:    -32601,
		Message: "Method not found",
		Data: map[string]string{
			"method": method,
		},
	})
}

func jsonParseError(w http.ResponseWriter, err error) {
	jsonError(w, rpc.Error{
		Code:    -32700,
		Message: "Parse error",
		Data: map[string]string{
			"error": err.Error(),
		},
	})
}

func jsonResult(w http.ResponseWriter, r interface{}) {
	bs, err := json.Marshal(r)
	if err != nil {
		jsonError(w, rpc.Error{
			Code:    -32603,
			Message: "Internal Error",
		})
		return
	}

	res := json.RawMessage(bs)
	jsonResponse(w, rpc.Response{
		Result: &res,
	})
}

func jsonResponse(w http.ResponseWriter, r rpc.Response) {
	bs, err := json.Marshal(response{
		Response: r,
		Body: rpc.Body{
			ID:      nextResponseID(),
			Version: "2.0",
		},
	})
	if err != nil {
		http.Error(w, "JSON serialization error", 500)
		return
	}
	if _, err := w.Write(bs); err != nil {
		klog.Error(err)
	}
}

func jsonRequest(w http.ResponseWriter, r *http.Request) (*rpc.Request, error) {
	req := &rpc.Request{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		jsonParseError(w, err)
		return nil, fmt.Errorf("Error decoding JSON RPC request: %w", err)
	}

	return req, nil
}
