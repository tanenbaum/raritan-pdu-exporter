package main

import (
	"fmt"
	"net/http"

	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
	"k8s.io/klog/v2"
)

const NumOutlets = 8

func pduHandler(w http.ResponseWriter, r *http.Request) {
	req, err := jsonRequest(w, r)
	if err != nil {
		klog.Error(err)
		return
	}

	switch method := req.Method; method {
	case "getMetaData":
		raritanResultJSON(w, raritan.PDUMetadata{
			Nameplate: raritan.PDUNameplate{
				Manufacturer: "Fake Manufacturer",
				Model:        "Fake Model",
				PartNumber:   "Fake Part Number",
				SerialNumber: "FAKESERIALNUMBER",
			},
			CtrlBoardSerial:      "FAKECTRLBOARDSERIAL",
			HasMeteredOutlets:    true,
			HasSwitchableOutlets: true,
			MacAddress:           "FAKEMACADDRESS",
		})
	case "getSettings":
		raritanResultJSON(w, raritan.PDUSettings{
			Name: "Fake Name",
		})
	case "getInlets":
		raritanResultJSON(w, []raritan.Resource{
			{
				RID:  "/model/inlet/0",
				Type: "Inlet_2_0_3",
			},
		})
	case "getOutlets":
		outlets := make([]raritan.Resource, NumOutlets)
		for i := 0; i < NumOutlets; i++ {
			outlets[i] = raritan.Resource{
				RID:  fmt.Sprintf("/model/outlet/%d", i),
				Type: "Outlet_2_1_4",
			}
		}
		raritanResultJSON(w, outlets)
	default:
		jsonMethodNotFound(w, method)
	}
}
