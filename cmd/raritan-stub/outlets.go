package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
	"k8s.io/klog/v2"
)

var outletSensors = func() map[string]func(id string) *raritan.Resource {
	numericSensor := "sensors.NumericSensor_4_0_2"
	stateSensor := "sensors.StateSensor_4_0_2"

	// nil value implies empty response
	sensors := map[string]*string{
		"voltage":                 &numericSensor,
		"current":                 &numericSensor,
		"peakCurrent":             &numericSensor,
		"maximumCurrent":          &numericSensor,
		"unbalancedCurrent":       &numericSensor,
		"activePower":             &numericSensor,
		"reactivePower":           &numericSensor,
		"apparentPower":           &numericSensor,
		"powerFactor":             &numericSensor,
		"displacementPowerFactor": nil,
		"activeEnergy":            &numericSensor,
		"apparentEnergy":          &numericSensor,
		"phaseAngle":              nil,
		"lineFrequency":           nil,
		"outletState":             &stateSensor,
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
				RID:  fmt.Sprintf("/model/outlet/%s/%s", id, n),
				Type: *t,
			}
		}
	}
	return ms
}()

func outletsHandler(w http.ResponseWriter, r *http.Request) {
	req, err := jsonRequest(w, r)
	if err != nil {
		klog.Error(err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	switch method := req.Method; method {
	case "getMetaData":
		raritanResultJSON(w, raritan.OutletMetadata{
			Label:          fmt.Sprintf("O%s", id),
			ReceptacleType: "Fake Receptacle Type",
		})
	case "getSettings":
		raritanResultJSON(w, raritan.OutletSettings{})
	case "getState":
		raritanResultJSON(w, raritan.OutletState{
			Available:  true,
			PowerState: 1,
		})
	case "getSensors":
		sens := make(map[string]*raritan.Resource, len(outletSensors))
		for k, v := range outletSensors {
			sens[k] = v(id)
		}
		raritanResultJSON(w, sens)
	default:
		jsonMethodNotFound(w, method)
	}
}
