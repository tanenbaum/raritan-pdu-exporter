package raritan

import (
	"fmt"
	"strings"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"
)

// Reading from Raritan sensor
type Reading struct {
	Timestamp uint
	Available bool
	Value     float64
}

func (c *Client) GetSensorReadings(sens []Resource) ([]Reading, error) {
	reqs := make([]bulkRequest, len(sens))
	for i, s := range sens {
		reqs[i] = bulkRequest{
			RID: s.RID,
			Request: rpc.Request{
				Method: sensorReadingMethod(s),
			},
			Return: &Reading{},
		}
	}

	if _, err := c.bulkCall(reqs); err != nil {
		return nil, err
	}

	rs := make([]Reading, len(sens))
	for i, r := range reqs {
		rd := r.Return.(*Reading)
		rs[i] = *rd
	}

	return rs, nil
}

// sensors have different reading methods based on type
func sensorReadingMethod(res Resource) string {
	if strings.Contains(res.Type, "NumericSensor") {
		return "getReading"
	} else if strings.Contains(res.Type, "StateSensor") {
		return "getState"
	}

	panic(fmt.Sprintf("Unknown Sensor type %s for resource %s", res.Type, res.RID))
}