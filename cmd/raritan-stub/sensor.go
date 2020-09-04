package main

import (
	"math/rand"
	"net/http"
	"time"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/raritan"
	"k8s.io/klog/v2"
)

func sensorHandler(w http.ResponseWriter, r *http.Request) {
	req, err := jsonRequest(w, r)
	if err != nil {
		klog.Error(err)
		return
	}

	switch method := req.Method; method {
	case "getReading":
		fallthrough
	case "getState":
		raritanResultJSON(w, raritan.Reading{
			Timestamp: uint(time.Now().Unix()),
			Available: true,
			Value:     rand.ExpFloat64(),
		})
	default:
		jsonMethodNotFound(w, method)
	}
}
