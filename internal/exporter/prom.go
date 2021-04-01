package exporter

import (
	"fmt"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/edgetic/oss/raritan-pdu-exporter/internal/raritan"
)

const (
	namespace = "pdu"
)

type PrometheusCollector struct {
	PDUInfo     raritan.PDUInfo
	logs        []SensorLog
	mux         sync.RWMutex
	metricNames func(string) string
}

func (c *PrometheusCollector) SetLogs(logs []SensorLog) {
	c.mux.Lock()
	c.logs = logs
	c.mux.Unlock()
}

func (c *PrometheusCollector) Describe(desc chan<- *prometheus.Desc) {}

func (c *PrometheusCollector) Collect(metric chan<- prometheus.Metric) {
	if c.metricNames == nil {
		c.metricNames = snakeCase()
	}

	labels := prometheus.Labels{
		"pdu_name":          c.PDUInfo.Name,
		"pdu_serial_number": c.PDUInfo.Nameplate.SerialNumber,
	}

	c.mux.RLock()
	defer c.mux.RUnlock()
	for _, l := range c.logs {
		metric <- prometheus.NewMetricWithTimestamp(l.Time,
			prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, strings.ToLower(l.Type), c.metricNames(l.Sensor)),
					fmt.Sprintf("%s sensor reading for %s", l.Type, l.Sensor), []string{"label"}, labels),
				prometheus.GaugeValue, l.Value, l.Label))
	}
}

func snakeCase() func(string) string {
	ms := map[string]string{}

	return func(v string) string {
		t, ok := ms[v]
		if !ok {
			sc := strcase.ToSnake(v)
			ms[v] = sc
			return sc
		}
		return t
	}
}
