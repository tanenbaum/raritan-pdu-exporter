package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
	"k8s.io/klog/v2"
)

var ocpSensors = func() map[string]func(id string) *raritan.Resource {
	numericSensor := "sensors.NumericSensor_4_0_2"
	overCurrentProtectorTripSensor := "pdumodel.OverCurrentProtectorTripSensor_1_0_5"

	// nil value implies empty response
	sensors := map[string]*string{
		"trip":    &overCurrentProtectorTripSensor,
		"voltage": nil,
		"current": &numericSensor,
	}

	ms := map[string]func(id string) *raritan.Resource{}
	for k, v := range sensors {
		n := k
		t := v
		ms[k] = func(id string) *raritan.Resource {
			if t == nil {
				return nil
			}
			return &raritan.Resource{
				RID:  fmt.Sprintf("/tfwopaque/%s/%s", id, n),
				Type: *t,
			}
		}
	}
	return ms
}()

func ocpHandler(w http.ResponseWriter, r *http.Request) {
	req, err := jsonRequest(w, r)
	if err != nil {
		klog.Error(err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	switch method := req.Method; method {
	case "getMetaData":
		raritanResultJSON(w, raritan.OCPMetadata{
			Label:      fmt.Sprintf("C%s", id),
			MaxTripCnt: 1000,
		})
	case "getSettings":
		raritanResultJSON(w, raritan.OCPSettings{})
	case "getSensors":
		sens := make(map[string]*raritan.Resource, len(ocpSensors))
		for k, v := range ocpSensors {
			sens[k] = v(id)
		}
		raritanResultJSON(w, sens)
	default:
		jsonMethodNotFound(w, method)
	}
}
