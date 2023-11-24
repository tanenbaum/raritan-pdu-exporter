package exporter

import (
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
)

const (
	namespace = "pdu"
)

type PrometheusCollector struct {
	Name     string
	PDUInfo  *raritan.PDUInfo
	SNMPINfo *raritan.SNMPInfo
	Labels   struct {
		UseConfigName   bool
		SerialNumber    bool
		SNMPSysContact  bool
		SNMPSysName     bool
		SNMPSydLocation bool
	}
	logs        []SensorLog
	mux         sync.RWMutex
	metricNames func(string) string
}

func (c *PrometheusCollector) SetLogs(logs []SensorLog) {
	c.mux.Lock()
	c.logs = logs
	c.mux.Unlock()
}

func (c *PrometheusCollector) SetPduInfo(pduInfo *raritan.PDUInfo) {
	c.mux.Lock()
	c.PDUInfo = pduInfo
	if !c.Labels.UseConfigName && c.PDUInfo != nil {
		c.Name = c.PDUInfo.Name
	}
	c.mux.Unlock()
}

func (c *PrometheusCollector) SetSnmpInfo(snmpInfo *raritan.SNMPInfo) {
	c.mux.Lock()
	c.SNMPINfo = snmpInfo
	c.mux.Unlock()
}

func (c *PrometheusCollector) Describe(desc chan<- *prometheus.Desc) {}

func (c *PrometheusCollector) Collect(metric chan<- prometheus.Metric) {
	if c.metricNames == nil {
		c.metricNames = snakeCase()
	}

	if c.PDUInfo != nil {
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "status", "pdu_active"),
			"PDU status",
			[]string{"pdu_name"},
			nil,
		)
		metric <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(1), c.Name)

		labels := prometheus.Labels{}
		if !c.Labels.UseConfigName {
			labels["pdu_name"] = c.PDUInfo.Name
		} else {
			labels["pdu_name"] = c.Name
		}
		if c.Labels.SerialNumber {
			labels["pdu_serial_number"] = c.PDUInfo.Nameplate.SerialNumber
		}
		if c.SNMPINfo != nil && c.Labels.SNMPSysName {
			labels["snmp_sys_name"] = c.SNMPINfo.SysName
		}
		if c.SNMPINfo != nil && c.Labels.SNMPSydLocation {
			labels["snmp_sys_location"] = c.SNMPINfo.SysLocation
		}
		if c.SNMPINfo != nil && c.Labels.SNMPSysContact {
			labels["snmp_sys_contact"] = c.SNMPINfo.SysContact
		}

		c.mux.RLock()
		defer c.mux.RUnlock()
		for _, l := range c.logs {
			help := fmt.Sprintf("%s sensor reading for %s", l.Type, l.Sensor)
			fqName := prometheus.BuildFQName(namespace, strings.ToLower(l.Type), c.metricNames(l.Sensor))
			metric <- prometheus.NewMetricWithTimestamp(l.Time,
				prometheus.MustNewConstMetric(
					prometheus.NewDesc(fqName, help, []string{"label"}, labels),
					prometheus.GaugeValue, l.Value, l.Label,
				),
			)
		}
	} else {
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "status", "pdu_active"),
			"PDU status",
			[]string{"pdu_name"},
			nil,
		)
		metric <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(0), c.Name)
	}

}

func (c *PrometheusCollector) Match(patterns []string) bool {
	return matchAnyFilter(c.Name, patterns)
}
