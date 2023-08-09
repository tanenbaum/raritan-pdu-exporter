package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/exporter"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/raritan"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/rpc"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// Config for
type CliConfig struct {
	Name       string `short:"n" long:"name" description:"Name of the endpoint. The default is the hostname. (only relevant in pool mode)"`
	Address    string `short:"a" long:"address" description:"Address of the PDU JSON RPC endpoint"`
	Timeout    int    `long:"timeout" default:"10" description:"Timeout of PDU RPC requests in seconds"`
	Username   string `short:"u" long:"username" env:"PDU_USERNAME" description:"Username for PDU access"`
	Password   string `short:"p" long:"password" env:"PDU_PASSWORD" description:"Password for PDU access"`
	Metrics    bool   `long:"metrics" description:"Enable prometheus metrics endpoint"`
	Port       uint   `long:"port" default:"2112" description:"Prometheus metrics port"`
	Interval   uint   `short:"i" long:"interval" default:"10" description:"Interval between data scrapes"`
	Mode       string `short:"m" long:"mode" default:"single" description:"operation mode [single or pool]"`
	ConfigPath string `short:"c" long:"config" value-name:"FILE" description:"path to pool config"`
}

// func (cc *CliConfig) Url() string {
// 	if cc.Address == "" {
// 		return ""
// 	} else if !strings.HasPrefix(cc.Address, "https://") || !strings.HasPrefix(cc.Address, "http://") {
// 		if strings.Contains(cc.Address, "443") {
// 			return "https://" + cc.Address
// 		} else {
// 			return "http://" + cc.Address
// 		}
// 	}
// 	return cc.Address
// }

type FileConfig struct {
	// Address   string      `json:"address" yaml:"address"`
	Timeout   int         `json:"timeout" yaml:"timeout"`
	Username  string      `json:"username" yaml:"username"`
	Password  string      `json:"password" yaml:"password"`
	Metrics   bool        `json:"metrics" yaml:"metrics"`
	Port      uint        `json:"port" yaml:"port"`
	Interval  uint        `json:"interval" yaml:"interval"`
	Mode      string      `json:"mode" yaml:"mode"`
	PduConfig []PduConfig `json:"pdu_config" yaml:"pdu_config"`
}

type Config struct {
	Metrics   bool                 `json:"metrics" yaml:"metrics"`
	Port      uint                 `json:"port" yaml:"port"`
	Interval  uint                 `json:"interval" yaml:"interval"`
	Mode      string               `json:"mode" yaml:"mode"`
	PduConfig map[string]PduConfig `json:"pdu_config" yaml:"pdu_config"`
}

type PduConfig struct {
	name     string `json:"name" yaml:"name"`
	Address  string `json:"address" yaml:"address"`
	Timeout  int    `json:"timeout" yaml:"timeout"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

func (c *PduConfig) Name() string {
	if c.name != "" {
		return c.name
	} else {
		url, err := url.Parse(c.Url())
		if err != nil {
			return ""
		}
		return url.Hostname()
	}
}

func (cc *PduConfig) Url() string {
	if cc.Address == "" {
		return ""
	} else if !strings.HasPrefix(cc.Address, "https://") || !strings.HasPrefix(cc.Address, "http://") {
		if strings.Contains(cc.Address, "443") {
			return "https://" + cc.Address
		} else {
			return "http://" + cc.Address
		}
	}
	return cc.Address
}

func (cliConf *CliConfig) GetConfig() (*Config, error) {
	conf := &Config{
		PduConfig: map[string]PduConfig{},
	}

	if cliConf.Address != "" && cliConf.Username != "" && cliConf.Password != "" {
		pduConfig := PduConfig{
			name:     cliConf.Name,
			Address:  cliConf.Address,
			Username: cliConf.Username,
			Password: cliConf.Password,
			Timeout:  cliConf.Timeout,
		}
		conf.PduConfig[pduConfig.Name()] = pduConfig
	}

	if cliConf.ConfigPath != "" {
		fileConfig, err := ReadConfigFromFile(cliConf.ConfigPath)
		if err != nil {
			return nil, err
		}
		fmt.Printf("fileConfig: %+v\n", fileConfig)
		fmt.Printf("cliConfig: %+v\n", cliConf)

		for _, pduConf := range fileConfig.PduConfig {
			fmt.Printf("pduConfig: %+v\n", pduConf)
			var username, password string
			var timeout int

			if pduConf.Username != "" {
				username = pduConf.Username
			} else if fileConfig.Username != "" {
				username = fileConfig.Username
			} else if cliConf.Username != "" {
				username = cliConf.Username
			}

			if pduConf.Password != "" {
				password = pduConf.Password
			} else if fileConfig.Password != "" {
				password = fileConfig.Password
			} else if cliConf.Password != "" {
				password = cliConf.Password
			}

			if pduConf.Timeout != 0 {
				timeout = pduConf.Timeout
			} else if fileConfig.Timeout != 0 {
				timeout = fileConfig.Timeout
			} else if cliConf.Timeout != 0 {
				timeout = cliConf.Timeout
			}

			conf.PduConfig[pduConf.Name()] = PduConfig{
				name:     pduConf.name,
				Address:  pduConf.Address,
				Username: username,
				Password: password,
				Timeout:  timeout,
			}
		}
		conf.Metrics = fileConfig.Metrics
		conf.Interval = fileConfig.Interval
		conf.Mode = fileConfig.Mode
		conf.Port = fileConfig.Port
	}

	conf.Metrics = conf.Metrics || cliConf.Metrics
	if conf.Port == 0 && cliConf.Port != 0 {
		conf.Port = cliConf.Port
	}
	if conf.Interval == 0 && cliConf.Interval != 0 {
		conf.Interval = cliConf.Interval
	}
	if conf.Mode == "" && cliConf.Mode != "" {
		conf.Mode = cliConf.Mode
	}
	fmt.Printf("conf: %+v\n", conf)
	return conf, nil
}

func ReadConfigFromFile(confPath string) (*FileConfig, error) {
	fc := &FileConfig{}

	content, err := os.ReadFile(confPath)
	if err != nil {
		log.Printf("[error] %v\n", err)
	}
	if strings.HasSuffix(confPath, ".yaml") || strings.HasSuffix(confPath, ".yml") {
		err := yaml.Unmarshal(content, fc)
		if err != nil {
			log.Printf("[error] %v\n", err)
		}
	} else if strings.HasSuffix(confPath, ".json") {
		err := json.Unmarshal(content, fc)
		if err != nil {
			log.Printf("[error] %v\n", err)
		}
	}

	return fc, nil
}

func Execute() {
	cltrs = make(map[string]*exporter.PrometheusCollector)
	args := os.Args

	klogFs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(klogFs)
	conf := &CliConfig{}

	p := flags.NewParser(conf, flags.Default|flags.IgnoreUnknown)

	fs, err := p.ParseArgs(args)
	if err != nil {
		if _, ok := err.(*flags.Error); !ok {
			klog.Exitf("Error parsing args: %v", err)
		}
		return
	}

	klogFs.Parse(fs)

	fmt.Printf("%+v\n", conf)

	c, err := conf.GetConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}
	klog.Infof("Server config: port=%d metrics=%t\n", c.Port, c.Metrics)
	for _, p := range c.PduConfig {
		klog.Infof("PDU config: name=%s url=%s\n", p.Name(), p.Url())
	}

	ExporterPool(c)
}

func ExporterSingle(conf *CliConfig) {
	baseURL, err := url.Parse(conf.Address)
	if err != nil {
		klog.Exitf("Error parsing URL: %v", err)
	}

	q := raritan.Client{
		RPCClient: rpc.NewClient(time.Duration(conf.Timeout)*time.Second, rpc.Auth{
			Username: conf.Username,
			Password: conf.Password,
		}),
		BaseURL: *baseURL,
	}

	go metrics(*conf)

	res, err := q.GetPDUInfo()
	if err != nil {
		klog.Exitf("Error getting PDU info: %v", err)
	}
	klog.V(1).Infof("PDU Info: %+v", res)

	collector := &exporter.PrometheusCollector{
		PDUInfo: *res,
	}
	prometheus.MustRegister(collector)

	ctx, cf := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		cf()
	}()

	ls, err := exporter.Run(ctx, q, conf.Interval)
	if err != nil {
		klog.Exit(err)
	}

	go func() {
		for l := range ls {
			klog.Infof("%v", l)
			collector.SetLogs(l)
		}
	}()

	<-ctx.Done()
}

var cltrs map[string]*exporter.PrometheusCollector

func ExporterPool(conf *Config) {

	ctx, cf := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		cf()
	}()

	go metricsPool(*conf)

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

		res, err := q.GetPDUInfo()
		if err != nil {
			klog.Exitf("Error getting PDU info: %v", err)
		}
		klog.V(1).Infof("PDU Info: %+v", res)

		collector := &exporter.PrometheusCollector{
			PDUInfo: *res,
		}
		prometheus.MustRegister(collector)

		ls, err := exporter.Run(ctx, q, conf.Interval)
		if err != nil {
			klog.Exit(err)
		}

		go func() {
			for l := range ls {
				// klog.Infof("%v", l)
				collector.SetLogs(l)
			}
		}()
		cltrs[pduConf.Name()] = collector
	}

	<-ctx.Done()
}

func metrics(c CliConfig) {
	if !c.Metrics {
		return
	}

	klog.V(1).Infof("Starting Prometheus metrics server on %d", c.Port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `PDU Metrics are at <a href="/metrics">/metrics<a>`)
	})
	http.HandleFunc("/metrics", metricsHandler)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil); err != nil {
		klog.Errorf("HTTP server error: %v", err)
	}
}

func metricsPool(c Config) {
	if !c.Metrics {
		return
	}

	klog.V(1).Infof("Starting Prometheus metrics server on %d", c.Port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `PDU Metrics are at <a href="/metrics">/metrics<a>`)
	})
	http.HandleFunc("/metrics", metricsHandler)
	// http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil); err != nil {
		klog.Errorf("HTTP server error: %v", err)
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	endpoint := params.Get("endpoint")
	if endpoint == "" {
		http.Error(w, "Endpoint parameter is missing", http.StatusBadRequest)
		return
	}
	for k, v := range params {
		fmt.Printf("%s: %v\n", k, v)
	}

	registry := prometheus.NewRegistry()

	if col, ok := cltrs[endpoint]; ok {
		registry.MustRegister(col)
	} else if len(cltrs) < 2 || strings.ToLower(endpoint) == "all" || strings.ToLower(endpoint) == "*" {
		for _, c := range cltrs {
			registry.MustRegister(c)
		}
	} else {
		http.Error(w, "endpoint not found", http.StatusNotFound)
		return
	}
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}
