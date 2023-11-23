package raritan

import "github.com/tanenbaum/raritan-pdu-exporter/internal/rpc"

var (
	snmpPath = mustURL("/snmp")
)

// SNMP Info for PDU
type SNMPInfo struct {
	SNMPConfiguration
}

// SNMP Configuration
type SNMPConfiguration struct {
	ReadComm    string
	SysContact  string
	SysLocation string
	SysName     string
	V2Enabled   bool
	V3Enabled   bool
	WriteComm   string
}

// GetSNMPInfo returns SNMP info
func (c *Client) GetSNMPInfo() (*SNMPInfo, error) {
	snmpConfig := &SNMPConfiguration{}
	reqs := []bulkRequest{
		{
			RID: snmpPath.String(),
			Request: rpc.Request{
				Method: "getConfiguration",
			},
			Return: snmpConfig,
		},
	}

	if _, err := c.bulkCall(reqs); err != nil {
		return nil, err
	}
	return &SNMPInfo{
		SNMPConfiguration: *snmpConfig,
	}, nil
}
