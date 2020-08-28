package raritan

import "gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"

// InletInfo for PDU inlet
type InletInfo struct {
	Resource
	InletMetadata
	InletSettings
	InletSensors
}

// InletMetadata metadata
type InletMetadata struct {
	Label    string
	PlugType string
}

// InletSettings containing name
type InletSettings struct {
	Name string
}

// InletSensors - map string -> RID - nil resource is not provided
type InletSensors = map[string]*Resource

func (c *Client) GetInletsInfo(ins []Resource) ([]InletInfo, error) {
	reqs := make([]bulkRequest, len(ins)*3)
	for i, in := range ins {
		i *= 3
		reqs[i] = bulkRequest{
			RID: in.RID,
			Request: rpc.Request{
				Method: "getMetaData",
			},
			Return: &InletMetadata{},
		}
		reqs[i+1] = bulkRequest{
			RID: in.RID,
			Request: rpc.Request{
				Method: "getSettings",
			},
			Return: &InletSettings{},
		}
		reqs[i+2] = bulkRequest{
			RID: in.RID,
			Request: rpc.Request{
				Method: "getSensors",
			},
			Return: &InletSensors{},
		}
	}
	if _, err := c.bulkCall(reqs); err != nil {
		return nil, err
	}

	infos := make([]InletInfo, len(ins))
	for i, in := range ins {
		j := i * 3
		meta := reqs[j].Return.(*InletMetadata)
		sett := reqs[j+1].Return.(*InletSettings)
		sens := reqs[j+2].Return.(*InletSensors)
		infos[i] = InletInfo{
			Resource:      in,
			InletMetadata: *meta,
			InletSettings: *sett,
			InletSensors:  *sens,
		}
	}
	return infos, nil
}
