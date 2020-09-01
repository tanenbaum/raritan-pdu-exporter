package scraper

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/raritan"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

func Run(client raritan.Client, interval uint) error {
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

	sens := allSensors(insInfo, olsInfo)

	wait.UntilWithContext(context.TODO(), func(ctx context.Context) {
		ss, err := client.GetSensorReadings(sens)
		if err != nil {
			klog.Errorf("Error getting sensor data: %v", err)
			return
		}
		klog.V(1).Infof("All sensors: %+v", ss)
	}, time.Second*time.Duration(interval))

	return nil
}

func allSensors(iis []raritan.InletInfo, ois []raritan.OutletInfo) []raritan.Resource {
	sens := []raritan.Resource{}
	for _, i := range iis {
		for _, v := range i.InletSensors {
			sens = append(sens, *v)
		}
	}
	for _, o := range ois {
		for _, v := range o.OutletSensors {
			sens = append(sens, *v)
		}
	}
	return sens
}
