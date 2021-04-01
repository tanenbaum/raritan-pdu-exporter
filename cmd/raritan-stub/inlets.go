package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.com/edgetic/oss/raritan-pdu-exporter/internal/raritan"
	"k8s.io/klog/v2"
)

var inletSensors = func() map[string]func(id string) *raritan.Resource {
	numericSensor := "sensors.NumericSensor_4_0_2"
	stateSensor := "sensors.StateSensor_4_0_2"
	residualSensor := "ResidualCurrentStateSensor_2_0_2"

	// nil value implies empty response
	sensors := map[string]*string{
		"voltage":                 &numericSensor,
		"current":                 &numericSensor,
		"peakCurrent":             &numericSensor,
		"residualCurrent":         &numericSensor,
		"residualDCCurrent":       nil,
		"activePower":             &numericSensor,
		"reactivePower":           &numericSensor,
		"apparentPower":           &numericSensor,
		"powerFactor":             &numericSensor,
		"displacementPowerFactor": nil,
		"activeEnergy":            &numericSensor,
		"apparentEnergy":          &numericSensor,
		"unbalancedCurrent":       &numericSensor,
		"lineFrequency":           &numericSensor,
		"phaseAngle":              nil,
		"powerQuality":            &stateSensor,
		"surgeProtectorStatus":    &stateSensor,
		"residualCurrentStatus":   &residualSensor,
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
				RID:  fmt.Sprintf("/model/inlet/%s/%s", id, n),
				Type: *t,
			}
		}
	}
	return ms
}()

func inletsHandler(w http.ResponseWriter, r *http.Request) {
	req, err := jsonRequest(w, r)
	if err != nil {
		klog.Error(err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	switch method := req.Method; method {
	case "getMetaData":
		raritanResultJSON(w, raritan.InletMetadata{
			Label:    fmt.Sprintf("I%s", id),
			PlugType: "Fake Plug Type",
		})
	case "getSettings":
		raritanResultJSON(w, raritan.InletSettings{})
	case "getSensors":
		sens := make(map[string]*raritan.Resource, len(inletSensors))
		for k, v := range inletSensors {
			sens[k] = v(id)
		}
		raritanResultJSON(w, sens)
	default:
		jsonMethodNotFound(w, method)
	}
}
