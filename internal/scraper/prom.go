package scraper

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/edgetic/hw/pdu-sensors/internal/raritan"
)

type PrometheusCollector struct {
	PDUInfo raritan.PDUInfo
	logs    []SensorLog
	mux     sync.RWMutex
}

func (c *PrometheusCollector) SetLogs(logs []SensorLog) {
	c.mux.Lock()
	c.logs = logs
	c.mux.Unlock()
}

func (c *PrometheusCollector) Describe(desc chan<- *prometheus.Desc) {
	desc <- prometheus.NewDesc("pdu", "", nil, prometheus.Labels{
		"pdu_name":          c.PDUInfo.Name,
		"pdu_serial_number": c.PDUInfo.Nameplate.SerialNumber,
	})
}

func (c *PrometheusCollector) Collect(metric chan<- prometheus.Metric) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	for _, l := range c.logs {
		metric <- prometheus.NewMetricWithTimestamp(l.Time,
			prometheus.MustNewConstMetric(
				prometheus.NewDesc(fmt.Sprintf("%s_%s", l.Type, l.Sensor), fmt.Sprintf("%s sensor reading for %s", l.Type, l.Sensor), []string{"label"}, nil), prometheus.GaugeValue, l.Value, l.Label))
	}
}
