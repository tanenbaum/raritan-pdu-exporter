package main

import (
	"net/http"

	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
	"k8s.io/klog/v2"
)

func snmpHandler(w http.ResponseWriter, r *http.Request) {
	req, err := jsonRequest(w, r)
	if err != nil {
		klog.Error(err)
		return
	}

	switch method := req.Method; method {
	case "getConfiguration":
		raritanResultJSON(w, raritan.SNMPConfiguration{
			ReadComm:    "ReadComm",
			SysContact:  "SysContact",
			SysLocation: "SysLocation",
			SysName:     "SysName",
			V2Enabled:   true,
			V3Enabled:   false,
			WriteComm:   "WriteComm",
		})
	default:
		jsonMethodNotFound(w, method)
	}
}
