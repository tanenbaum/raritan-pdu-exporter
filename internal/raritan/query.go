package raritan

import (
	"encoding/json"
	"errors"
	"net/url"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"
)

var (
	pduPath = mustURL("/model/pdu/0")
)

// Client to query Raritan PDU
type Client struct {
	RPCClient rpc.Client
	BaseURL   url.URL
}

type result struct {
	Return json.RawMessage `json:"_ret_"`
}

// GetPDUMetadata returns metadata for main PDU entry
func (c *Client) GetPDUMetadata() (interface{}, error) {
	ret := map[string]interface{}{}
	if _, err := c.call(*c.BaseURL.ResolveReference(&pduPath), rpc.Request{
		Method: "getMetaData",
	}, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) call(url url.URL, req rpc.Request, ret interface{}) (*result, error) {
	r, err := c.RPCClient.Call(url, req)
	if err != nil {
		return nil, err
	}
	if r.IsError() {
		return nil, r.Error
	}
	if r.Result == nil {
		return nil, errors.New("Expected RPC result not nil")
	}
	res := &result{}
	if err := json.Unmarshal(*r.Result, res); err != nil {
		return nil, err
	}
	if ret != nil {
		if err := json.Unmarshal(res.Return, ret); err != nil {
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
