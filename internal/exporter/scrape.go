package exporter

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/raritan"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

type SensorLog struct {
	Type     string
	Label    string
	Sensor   string
	Time     time.Time
	Value    float64
	Resource raritan.Resource
}

func (l SensorLog) String() string {
	return fmt.Sprintf("%s: %s, sensor: %s, val: %f, unix: %d", l.Type, l.Label, l.Sensor, l.Value, l.Time.Unix())
}

func Run(ctx context.Context, client raritan.Client, interval uint) (<-chan []SensorLog, error) {
	sc := make(chan []SensorLog, 1)
	if err := pollSensors(client, sc); err != nil {
		return nil, err
	}

	// poll sensors every 10 x interval
	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		if err := pollSensors(client, sc); err != nil {
			klog.Errorf("Error polling sensors: %v", err)
			return
		}
	}, time.Second*time.Duration(interval*10))

	// sensor length might change over time - best guess here
	// consumer should outpace the polling anyway
	log := make(chan []SensorLog)
	sens := <-sc

	go wait.UntilWithContext(ctx, func(_ context.Context) {
		// Check for new sensors
		select {
		case sens = <-sc:
		default:
		}

		logs, err := pollReadings(client, sens)
		if err != nil {
			klog.Errorf("Error polling readings: %v", err)
		}
		log <- logs
	}, time.Second*time.Duration(interval))

	return log, nil
}

func pollSensors(client raritan.Client, sl chan<- []SensorLog) error {
	ins, err := client.GetPDUInlets()
	if err != nil {
		return fmt.Errorf("Error requesting PDU Inlets: %w", err)
	}

	insInfo, err := client.GetInletsInfo(ins)
	if err != nil {
		return fmt.Errorf("Error getting Inlet info: %w", err)
	}

	klog.V(1).Infof("PDU Inlets: %+v", insInfo)

	ols, err := client.GetPDUOutlets()
	if err != nil {
		return fmt.Errorf("Error requesting PDU Outlets: %w", err)
	}

	olsInfo, err := client.GetOutletsInfo(ols)
	if err != nil {
		return fmt.Errorf("Error getting Outlet info: %w", err)
	}

	klog.V(1).Infof("PDU Outlets: %+v", olsInfo)

	sl <- allSensors(insInfo, olsInfo)
	return nil
}

func pollReadings(client raritan.Client, sens []SensorLog) ([]SensorLog, error) {
	res := make([]raritan.Resource, len(sens))
	for i, s := range sens {
		res[i] = s.Resource
	}

	rs, err := client.GetSensorReadings(res)
	if err != nil {
		return nil, fmt.Errorf("Error getting sensor data: %w", err)
	}
	logs := []SensorLog{}
	for i, r := range rs {
		s := sens[i]
		if !r.Available {
			klog.V(4).Infof("Sensor %s not available", s.Resource.RID)
			continue
		}

		s.Time = time.Unix(int64(r.Timestamp), 0)
		s.Value = r.Value
		logs = append(logs, s)
	}
	return logs, nil
}

func allSensors(iis []raritan.InletInfo, ois []raritan.OutletInfo) []SensorLog {
	sens := []SensorLog{}
	for _, i := range iis {
		for k, v := range i.Sensors {
			label := i.Label
			if i.Name != "" {
				label = i.Name
			}
			sens = append(sens, SensorLog{
				Resource: v,
				Sensor:   k,
				Type:     "inlet",
				Label:    label,
			})
		}
	}
	for _, o := range ois {
		for k, v := range o.Sensors {
			label := o.Label
			if o.Name != "" {
				label = o.Name
			}
			sens = append(sens, SensorLog{
				Resource: v,
				Sensor:   k,
				Type:     "outlet",
				Label:    label,
			})
		}
	}
	return sens
}
