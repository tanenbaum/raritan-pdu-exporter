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
	"regexp"
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
	Name       string `short:"n" long:"name" env:"PDU_NAME" description:"Name of the endpoint. Only relevant with multiple endpoints. (default: <name from pdu>)"`
	Address    string `short:"a" long:"address" env:"PDU_ADDRESS" description:"Address of the PDU JSON RPC endpoint"`
	Timeout    int    `long:"timeout" default:"10" description:"Timeout of PDU RPC requests in seconds"`
	Username   string `short:"u" long:"username" env:"PDU_USERNAME" description:"Username for PDU access"`
	Password   string `short:"p" long:"password" env:"PDU_PASSWORD" description:"Password for PDU access"`
	Metrics    bool   `long:"metrics" description:"Enable prometheus metrics endpoint"`
	Port       uint   `long:"port" default:"2112" description:"Prometheus metrics port"`
	Interval   uint   `short:"i" long:"interval" default:"10" description:"Interval between data scrapes"`
	ConfigPath string `short:"c" long:"config" value-name:"FILE" description:"path to pool config"`
}

type FileConfig struct {
	// Address   string      `json:"address" yaml:"address"`
	Timeout   int         `json:"timeout" yaml:"timeout"`
	Username  string      `json:"username" yaml:"username"`
	Password  string      `json:"password" yaml:"password"`
	Metrics   bool        `json:"metrics" yaml:"metrics"`
	Port      uint        `json:"port" yaml:"port"`
	Interval  uint        `json:"interval" yaml:"interval"`
	PduConfig []PduConfig `json:"pdu_config" yaml:"pdu_config"`
}

type Config struct {
	Metrics   bool        `json:"metrics" yaml:"metrics"`
	Port      uint        `json:"port" yaml:"port"`
	Interval  uint        `json:"interval" yaml:"interval"`
	PduConfig []PduConfig `json:"pdu_config" yaml:"pdu_config"`
}

type PduConfig struct {
	Name     string `json:"name" yaml:"name"`
	Address  string `json:"address" yaml:"address"`
	Timeout  int    `json:"timeout" yaml:"timeout"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// func (c *PduConfig) Name() string {
// 	if c.PduName != "" {
// 		return c.PduName
// 	} else {
// 		url, err := url.Parse(c.Url())
// 		if err != nil {
// 			return ""
// 		}
// 		return url.Hostname()
// 	}
// }

func (cc *PduConfig) Url() string {
	if cc.Address == "" {
		return ""
	} else if !strings.HasPrefix(cc.Address, "https://") && !strings.HasPrefix(cc.Address, "http://") {
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
		PduConfig: []PduConfig{},
	}

	if cliConf.Address != "" && cliConf.Username != "" && cliConf.Password != "" {
		pduConfig := PduConfig{
			Name:     cliConf.Name,
			Address:  cliConf.Address,
			Username: cliConf.Username,
			Password: cliConf.Password,
			Timeout:  cliConf.Timeout,
		}
		conf.PduConfig = append(conf.PduConfig, pduConfig)
	}

	if cliConf.ConfigPath != "" {
		fileConfig, err := ReadConfigFromFile(cliConf.ConfigPath)
		if err != nil {
			return nil, err
		}

		for _, pduConf := range fileConfig.PduConfig {

			if pduConf.Username == "" {
				if fileConfig.Username != "" {
					pduConf.Username = fileConfig.Username
				} else if cliConf.Username != "" {
					pduConf.Username = cliConf.Username
				}
			}

			if pduConf.Password == "" {
				if fileConfig.Password != "" {
					pduConf.Password = fileConfig.Password
				} else if cliConf.Password != "" {
					pduConf.Password = cliConf.Password
				}
			}

			if pduConf.Timeout == 0 {
				if fileConfig.Timeout != 0 {
					pduConf.Timeout = fileConfig.Timeout
				} else if cliConf.Timeout != 0 {
					pduConf.Timeout = cliConf.Timeout
				}
			}

			conf.PduConfig = append(conf.PduConfig, pduConf)
		}
		conf.Metrics = fileConfig.Metrics
		conf.Interval = fileConfig.Interval
		conf.Port = fileConfig.Port
	}

	conf.Metrics = conf.Metrics || cliConf.Metrics
	if conf.Port == 0 && cliConf.Port != 0 {
		conf.Port = cliConf.Port
	}
	if conf.Interval == 0 && cliConf.Interval != 0 {
		conf.Interval = cliConf.Interval
	}

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

var cltrs map[string]*exporter.PrometheusCollector

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

	c, err := conf.GetConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}
	klog.Infof("Server config: port=%d metrics=%t\n", c.Port, c.Metrics)
	for _, p := range c.PduConfig {
		var name string
		if p.Name != "" {
			name = p.Name
		} else {
			name = "<get name from pdu>"
		}
		klog.Infof("PDU config: name=%s url=%s\n", name, p.Url())
	}

	Exporter(c)
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

		res, err := q.GetPDUInfo()
		if err != nil {
			klog.Errorf("Error getting PDU info: %v", err)
			klog.Errorf("Skipping %s", pduConf.Address)
			continue
		}
		klog.V(1).Infof("PDU Info: %+v", res)

		collector := &exporter.PrometheusCollector{
			PDUInfo: *res,
		}

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
		var name string
		if pduConf.Name == "" {
			name = res.Name
		} else {
			name = pduConf.Name
		}
		cltrs[name] = collector
	}

	<-ctx.Done()
}

func metrics(c Config) {
	if !c.Metrics {
		return
	}

	klog.V(1).Infof("Starting Prometheus metrics server on %d", c.Port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `PDU Metrics are at <a href="/metrics">/metrics<a>`)
	})
	http.HandleFunc("/metrics", metricsHandler)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil); err != nil {
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
	klog.Infof("%+v\n", endpointFilter)
	for name, collector := range cltrs {
		if matchAnyFilter(name, endpointFilter) || all {
			registry.MustRegister(collector)
		}
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func matchAnyFilter(a string, patternList []string) bool {
	for _, p := range patternList {
		reg := regexp.MustCompile("^" + strings.ReplaceAll(p, "*", ".*") + "$")
		if reg.MatchString(a) {
			fmt.Printf("%s does match %s\n", a, p)
			return true
		} else {
			fmt.Printf("%s does not match %s\n", a, p)
		}
	}
	return false
}

func listContains(lst []string, a string) bool {
	for _, p := range lst {
		if p == "all" {
			return true
		}
	}
	return false
}
