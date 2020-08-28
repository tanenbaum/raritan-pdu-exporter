package raritan

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"
)

var (
	bulkPath = mustURL("/bulk")
)

// Client to query Raritan PDU
type Client struct {
	RPCClient rpc.Client
	BaseURL   url.URL
}

type result struct {
	Return json.RawMessage `json:"_ret_"`
}

type bulkResult struct {
	Responses []bulkResponse
}

type bulkResponse struct {
	JSON     *rpc.Response
	StatCode int
}

type bulkRequest struct {
	// RID is resource path for bulk endpoint
	RID string
	// Request is body of request
	Request rpc.Request
	// Return type to be unmarshalled to, pointer
	Return interface{}
}

func (c *Client) call(url url.URL, req rpc.Request, ret interface{}) (*result, error) {
	r, err := c.RPCClient.Call(url, req)
	if err != nil {
		return nil, err
	}
	res := &result{}
	if err := unmarshallResult(r, res); err != nil {
		return nil, fmt.Errorf("Error unmarshalling result for %s method %s: %w", url.String(), req.Method, err)
	}
	if ret != nil {
		if err := json.Unmarshal(res.Return, ret); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func unmarshallResult(r *rpc.Response, ret interface{}) error {
	if r.IsError() {
		return r.Error
	}
	if r.Result == nil {
		return errors.New("Expected RPC result not nil")
	}

	if err := json.Unmarshal(*r.Result, ret); err != nil {
		return err
	}
	return nil
}

func (c *Client) bulkCall(br []bulkRequest) (*bulkResult, error) {
	reqs := make([]map[string]interface{}, len(br))
	for i, r := range br {
		reqs[i] = map[string]interface{}{
			"rid": r.RID,
			"json": map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  r.Request.Method,
				"params":  r.Request.Params,
				"id":      i,
			},
		}
	}

	r, err := c.RPCClient.Call(*c.BaseURL.ResolveReference(&bulkPath), rpc.Request{
		Method: "performBulk",
		Params: map[string]interface{}{
			"requests": reqs,
		},
	})
	if err != nil {
		return nil, err
	}
	res := &bulkResult{}
	if err := unmarshallResult(r, res); err != nil {
		return nil, err
	}

	for i, r := range res.Responses {
		if r.StatCode != 200 {
			return nil, fmt.Errorf("Bulk response code not 200: %d", r.StatCode)
		}
		req := br[i]
		if req.Return == nil {
			continue
		}

		res := &result{}
		if err := unmarshallResult(r.JSON, res); err != nil {
			return nil, fmt.Errorf("Error unmarshalling result for %s method %s: %w", req.RID, req.Request.Method, err)
		}
		if err := json.Unmarshal(res.Return, req.Return); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func mustURL(path string) url.URL {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return *u
}
