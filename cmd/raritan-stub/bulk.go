package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"gitlab.com/edgetic/oss/raritan-pdu-exporter/internal/rpc"
	"k8s.io/klog/v2"
)

type BulkRequests struct {
	Requests []BulkRequest
}

type BulkRequest struct {
	RID  string
	JSON rpc.Request
}

type bulkResult struct {
	Responses []bulkResponse
}

type bulkResponse struct {
	StatCode int
	JSON     *rpc.Response
}

func bulkHandler(c rpc.Client, port uint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := jsonRequest(w, r)
		if err != nil {
			klog.Error(err)
			return
		}

		if req.Method != "performBulk" {
			jsonMethodNotFound(w, req.Method)
			return
		}

		bulk := &BulkRequests{}
		if err := mapstructure.Decode(req.Params, bulk); err != nil {
			jsonParseError(w, err)
			return
		}

		baseURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", port))

		res := make([]bulkResponse, len(bulk.Requests))
		for i, r := range bulk.Requests {
			path, _ := url.Parse(r.RID)
			r, err := c.Call(*baseURL.ResolveReference(path), r.JSON)
			code := 200
			if err != nil {
				klog.Errorf("Error performing bulk call: %v", err)
				code = 500
			}
			res[i] = bulkResponse{
				StatCode: code,
				JSON:     r,
			}
		}

		jsonResult(w, bulkResult{
			Responses: res,
		})
	}
}
