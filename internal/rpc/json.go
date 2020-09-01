package rpc

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

// Client for calling JSON RPC endpoints
type Client interface {
	Call(url.URL, Request) (*Response, error)
	BatchCall(url.URL, []Request) ([]Response, error)
}

// Request RPC attributes
type Request struct {
	// Method for request
	Method string `json:"method"`
	// Params for request
	Params map[string]interface{} `json:"params,omitEmpty"`
}

// Response from RPC
type Response struct {
	// Error required if error occurred
	Error *Error `json:"error"`
	// Result json if successful - raw message so we can convert later
	Result *json.RawMessage
}

// Error from RPC response
type Error struct {
	// Code for error
	Code int `json:"code"`
	// Message for error
	Message string `json:"message"`
	// Data for error metadata
	Data interface{} `json:"data"`
}

func (e Error) Error() string {
	return fmt.Sprintf("RPC Error, Code: %d, \"%s\", Data: %v", e.Code, e.Message, e.Data)
}

// Auth settings for requests
type Auth struct {
	Username string
	Password string
}

type body struct {
	Version string `json:"jsonrpc"`
	ID      int    `json:"id"`
}

type request struct {
	Request
	body
}

type response struct {
	Response
	body
}

type client struct {
	httpClient *http.Client
	auth       Auth
}

var requestID int64 = 0

func nextRequestID() int {
	return int(atomic.AddInt64(&requestID, 1))
}

// IsSuccess when Result is set
func (r Response) IsSuccess() bool {
	return r.Result != nil
}

// IsError when Error is set
func (r Response) IsError() bool {
	return r.Error != nil
}

// NewClient returns a new JSON RPC client
func NewClient(timeout time.Duration, auth Auth) Client {
	return &client{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		auth: auth,
	}
}

func (c *client) Call(url url.URL, req Request) (*Response, error) {
	bs, err := json.Marshal(request{
		Request: req,
		body: body{
			Version: "2.0",
			ID:      nextRequestID(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error marshalling JSON: %w", err)
	}
	r, err := http.NewRequest("POST", url.String(), bytes.NewReader(bs))
	if err != nil {
		return nil, fmt.Errorf("Error creating JSON RPC request: %w", err)
	}
	r.SetBasicAuth(c.auth.Username, c.auth.Password)

	res, err := c.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("Error performing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("Non 200 status code in RPC response: %s", res.Status)
	}

	response := &Response{}
	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, fmt.Errorf("Error unmarshalling response: %w", err)
	}

	return response, nil
}

func (c *client) BatchCall(url url.URL, reqs []Request) ([]Response, error) {
	// Raritan RPC doesn't support the standardised batch call so I haven't written it
	panic("Not implemented")
}
