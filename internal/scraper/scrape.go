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

	go wait.UntilWithContext(context.TODO(), func(ctx context.Context) {
		sens := []raritan.Resource{}
		for _, v := range insInfo[0].InletSensors {
			if v != nil {
				sens = append(sens, *v)
			}
		}
		ss, err := client.GetSensorReadings(sens)
		if err != nil {
			klog.Errorf("Error getting inlet sensors: %v", err)
			return
		}
		klog.V(1).Infof("Inlet sensors: %+v", ss)
	}, time.Second*time.Duration(interval))

	wait.UntilWithContext(context.TODO(), func(ctx context.Context) {
		sens := []raritan.Resource{}
		for _, v := range olsInfo[0].OutletSensors {
			if v != nil {
				sens = append(sens, *v)
			}
		}
		ss, err := client.GetSensorReadings(sens)
		if err != nil {
			klog.Errorf("Error getting outlet sensors: %v", err)
			return
		}
		klog.V(1).Infof("Outlet sensors: %+v", ss)
	}, time.Second*time.Duration(interval))

	return nil
}
