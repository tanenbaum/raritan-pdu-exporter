package raritan

import "gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"

var (
	pduPath = mustURL("/model/pdu/0")
)

type PDUMetadata struct {
	Nameplate               PDUNameplate
	CtrlBoardSerial         string
	HwRevision              string
	FwRevision              string
	MacAddress              string
	HasSwitchableOutlets    bool
	HasMeteredOutlets       bool
	HasLatchingOutletRelays bool
	IsInlineMeter           bool
	IsEnergyPulseSupported  bool
}

type PDUNameplate struct {
	Manufacturer string
	Model        string
	PartNumber   string
	SerialNumber string
}

type PDUSettings struct {
	Name string
}

type PDUInfo struct {
	PDUMetadata
	PDUSettings
}

// Resource is generic resource holder for reference link
type Resource struct {
	// RID resource id
	RID string
	// Type string for resource
	Type string
}

// GetPDUInfo returns info for main PDU entry
func (c *Client) GetPDUInfo() (*PDUInfo, error) {
	meta := &PDUMetadata{}
	sett := &PDUSettings{}
	reqs := []bulkRequest{
		{
			RID: pduPath.String(),
			Request: rpc.Request{
				Method: "getMetaData",
			},
			Return: meta,
		},
		{
			RID: pduPath.String(),
			Request: rpc.Request{
				Method: "getSettings",
			},
			Return: sett,
		},
	}

	if _, err := c.bulkCall(reqs); err != nil {
		return nil, err
	}
	return &PDUInfo{
		PDUMetadata: *meta,
		PDUSettings: *sett,
	}, nil
}

func (c *Client) GetPDUInlets() ([]Resource, error) {
	ret := []Resource{}
	if _, err := c.call(*c.BaseURL.ResolveReference(&pduPath), rpc.Request{
		Method: "getInlets",
	}, &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) GetPDUOutlets() ([]Resource, error) {
	ret := []Resource{}
	if _, err := c.call(*c.BaseURL.ResolveReference(&pduPath), rpc.Request{
		Method: "getOutlets",
	}, &ret); err != nil {
		return nil, err
	}

	return ret, nil
}
