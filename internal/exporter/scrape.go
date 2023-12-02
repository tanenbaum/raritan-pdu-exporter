package exporter

import (
	"context"
	"fmt"
	"time"

	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
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

func Run(ctx context.Context, client raritan.Client, interval uint, pollForSNMP bool) (<-chan []SensorLog, <-chan *raritan.PDUInfo, <-chan *raritan.SNMPInfo, error) {
	sc := make(chan []SensorLog, 1)

	// poll sensors every 10 x interval
	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		if err := pollSensors(client, sc); err != nil {
			klog.Errorf("Error polling sensors for %s\n", client.BaseURL.String())
			sc <- nil
		}
	}, time.Second*time.Duration(interval*10))

	// sensor length might change over time - best guess here
	// consumer should outpace the polling anyway
	log := make(chan []SensorLog)
	cPduInfo := make(chan *raritan.PDUInfo)
	cSnmpInfo := make(chan *raritan.SNMPInfo)
	sens := <-sc

	go wait.UntilWithContext(ctx, func(_ context.Context) {
		// Check if PDU is online and refresh info
		if err := client.ConnectionCheck(); err != nil {
			klog.Errorf("%s\n", err)
			return
		}

		pduInfo, err := getPduInfo(client)
		cPduInfo <- pduInfo
		if err != nil {
			klog.Errorf("%s\n", err)
			return
		}

		if pollForSNMP {
			snmpInfo, err := getSnmpInfo(client)
			cSnmpInfo <- snmpInfo
			if err != nil {
				klog.Errorf("%s\n", err)
				return
			}
		}

		// Check for new sensors
		select {
		case sens = <-sc:
		default:
		}

		logs, err := pollReadings(client, sens)
		if err != nil {
			klog.Errorf("%s", err)
		}
		log <- logs
	}, time.Second*time.Duration(interval))

	return log, cPduInfo, cSnmpInfo, nil
}

func getSensorInfo(client raritan.Client) ([]raritan.InletInfo, []raritan.OutletInfo, []raritan.OCPInfo, error) {
	ins, err := client.GetPDUInlets()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error requesting PDU Inlets: %w", err)
	}

	insInfo, err := client.GetInletsInfo(ins)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting Inlet info: %w", err)
	}

	klog.V(1).Infof("PDU Inlets: %+v", insInfo)

	ols, err := client.GetPDUOutlets()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error requesting PDU Outlets: %w", err)
	}

	olsInfo, err := client.GetOutletsInfo(ols)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting Outlet info: %w", err)
	}

	ocp, err := client.GetPDUOCP()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error requesting PDU OverCurrentProtectors: %w", err)
	}

	ocpInfo, err := client.GetOCPInfo(ocp)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting OverCurrentProtectors info: %w", err)
	}

	klog.V(1).Infof("PDU OverCurrentProtectors: %+v", ocpInfo)
	return insInfo, olsInfo, ocpInfo, nil
}

func getSensors(client raritan.Client) ([]SensorLog, error) {
	iis, ois, ocp, err := getSensorInfo(client)
	if err != nil {
		return nil, err
	}
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
	for _, o := range ocp {
		for k, v := range o.Sensors {
			label := o.Label
			if o.Name != "" {
				label = o.Name
			}
			sens = append(sens, SensorLog{
				Resource: v,
				Sensor:   k,
				Type:     "ocp",
				Label:    label,
			})
		}
	}
	return sens, nil
}

func pollSensors(client raritan.Client, sl chan<- []SensorLog) error {
	sensors, err := getSensors(client)
	if err != nil {
		return err
	}
	sl <- sensors
	return nil
}

func pollReadings(client raritan.Client, sens []SensorLog) ([]SensorLog, error) {
	if sens == nil {
		return nil, fmt.Errorf("no sensors available for %s", client.BaseURL.String())
	}
	res := make([]raritan.Resource, len(sens))
	for i, s := range sens {
		res[i] = s.Resource
	}

	rs, err := client.GetSensorReadings(res)
	if err != nil {
		return nil, fmt.Errorf("error getting sensor data: %w", err)
	}
	logs := []SensorLog{}
	for i, r := range rs {
		s := sens[i]
		if !r.Available {
			klog.V(4).Infof("sensor %s not available", s.Resource.RID)
			continue
		}

		s.Time = time.Unix(int64(r.Timestamp), 0)
		s.Value = r.Value
		logs = append(logs, s)
	}
	return logs, nil
}

func getPduInfo(client raritan.Client) (*raritan.PDUInfo, error) {
	pduInfo, err := client.GetPDUInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get PDU info from %s: %s", client.BaseURL.String(), err)
	}

	return pduInfo, nil
}

func getSnmpInfo(client raritan.Client) (*raritan.SNMPInfo, error) {
	snmpInfo, err := client.GetSNMPInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get SNMP info from %s: %s", client.BaseURL.String(), err)
	}

	return snmpInfo, nil
}
