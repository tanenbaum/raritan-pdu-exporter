package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/exporter"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/rpc"
	"k8s.io/klog/v2"
)

var cltrs []*exporter.PrometheusCollector

func Run() {
	cltrs = []*exporter.PrometheusCollector{}
	config, err := LoadConfig(os.Args)
	if err != nil {
		klog.Exitf("%s", err)
	}

	Exporter(config)
}

func Exporter(conf *Config) {

	ctx, cf := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		cf()
	}()

	go metrics(*conf)

	for _, pduConf := range conf.PduConfig {
		baseURL, err := url.Parse(pduConf.Url())
		if err != nil {
			klog.Exitf("Error parsing URL: %v", err)
		}

		q := raritan.Client{
			RPCClient: rpc.NewClient(time.Duration(pduConf.Timeout)*time.Second, rpc.Auth{
				Username: pduConf.Username,
				Password: pduConf.Password,
			}),
			BaseURL: *baseURL,
		}

		collector := &exporter.PrometheusCollector{
			Name: pduConf.Name,
		}
		collector.Labels.UseConfigName = conf.ExporterLabels["use_config_name"]
		collector.Labels.SerialNumber = conf.ExporterLabels["serial_number"]
		collector.Labels.SNMPSysContact = conf.ExporterLabels["snmp_sys_contact"]
		collector.Labels.SNMPSysName = conf.ExporterLabels["snmp_sys_name"]
		collector.Labels.SNMPSydLocation = conf.ExporterLabels["snmp_sys_location"]

		enableSNMP := collector.Labels.SNMPSydLocation || collector.Labels.SNMPSysContact || collector.Labels.SNMPSysName

		ls, cPduInfo, cSnmpInfo, err := exporter.Run(ctx, q, conf.Interval, enableSNMP)
		if err != nil {
			klog.Errorf("failed to connect to %s, skipping pdu", pduConf.Name)
		}

		go func() {
			for l := range ls {
				collector.SetLogs(l)
			}
		}()
		go func() {
			for pduInfo := range cPduInfo {
				collector.SetPduInfo(pduInfo)
			}
		}()
		go func() {
			for snmpInfo := range cSnmpInfo {
				collector.SetSnmpInfo(snmpInfo)
			}
		}()
		cltrs = append(cltrs, collector)
	}

	<-ctx.Done()
}

func metrics(c Config) {
	if !c.Metrics {
		return
	}

	r := mux.NewRouter()
	r.Use(logMW)
	klog.V(1).Infof("Starting Prometheus metrics server on %d", c.Port)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `PDU Metrics are at <a href="/metrics">/metrics<a>`)
	})
	r.HandleFunc("/metrics", metricsHandler)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", c.Port), r); err != nil {
		klog.Errorf("HTTP server error: %v", err)
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	endpointFilter := []string{}
	if params.Has("endpoint") {
		endpointFilter = append(endpointFilter, params.Get("endpoint"))
	}

	for k, v := range params {
		if k == "endpoint[]" {
			endpointFilter = append(endpointFilter, v...)
		}
	}

	registry := prometheus.NewRegistry()
	all := listContains(endpointFilter, "all") || len(endpointFilter) == 0
	for _, collector := range cltrs {
		if all || collector.Match(endpointFilter) {
			registry.MustRegister(collector)
		}
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func logMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		klog.V(1).Infof("%s - %s (%s)", r.Method, r.URL.RequestURI(), r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
