package raritan

import "gitlab.com/edgetic/oss/raritan-pdu-exporter/internal/rpc"

// OutletInfo for PDU outlet
type OutletInfo struct {
	Resource
	OutletMetadata
	OutletSettings
	OutletState
	Sensors
}

// OutletMetadata metadata
type OutletMetadata struct {
	Label          string
	ReceptacleType string
}

// OutletSettings containing name
type OutletSettings struct {
	Name string
}

// OutletState indicating state
type OutletState struct {
	Available  bool
	PowerState uint
}

func (c *Client) GetOutletsInfo(os []Resource) ([]OutletInfo, error) {
	reqs := make([]bulkRequest, len(os)*4)
	for i, o := range os {
		i *= 4
		reqs[i] = bulkRequest{
			RID: o.RID,
			Request: rpc.Request{
				Method: "getMetaData",
			},
			Return: &OutletMetadata{},
		}
		reqs[i+1] = bulkRequest{
			RID: o.RID,
			Request: rpc.Request{
				Method: "getSettings",
			},
			Return: &OutletSettings{},
		}
		reqs[i+2] = bulkRequest{
			RID: o.RID,
			Request: rpc.Request{
				Method: "getState",
			},
			Return: &OutletState{},
		}
		reqs[i+3] = bulkRequest{
			RID: o.RID,
			Request: rpc.Request{
				Method: "getSensors",
			},
			Return: &map[string]*Resource{},
		}
	}
	if _, err := c.bulkCall(reqs); err != nil {
		return nil, err
	}

	infos := make([]OutletInfo, len(os))
	for i, in := range os {
		j := i * 4
		meta := reqs[j].Return.(*OutletMetadata)
		sett := reqs[j+1].Return.(*OutletSettings)
		stat := reqs[j+2].Return.(*OutletState)
		sens := reqs[j+3].Return.(*map[string]*Resource)
		infos[i] = OutletInfo{
			Resource:       in,
			OutletMetadata: *meta,
			OutletSettings: *sett,
			OutletState:    *stat,
			Sensors:        filterEmptySensors(*sens),
		}
	}
	return infos, nil
}
