package main

import (
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/jessevdk/go-flags"
	"k8s.io/klog/v2"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/raritan"
	"gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"
	"gitlab.com/edgetic/hw/pdu-sensors/internal/scraper"

	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config for scraper
type Config struct {
	Address  string `short:"a" long:"address" required:"true" description:"Address of the PDU JSON RPC endpoint"`
	Timeout  int    `long:"timeout" default:"10" description:"Timeout of PDU RPC requests in seconds"`
	Username string `short:"u" long:"username" env:"PDU_USERNAME" description:"Username for PDU access"`
	Password string `short:"p" long:"password" env:"PDU_PASSWORD" description:"Password for PDU access"`
	Metrics  bool   `long:"metrics" description:"Enable prometheus metrics endpoint"`
	Port     uint   `long:"port" default:"2112"`
	Interval uint   `short:"i" long:"interval" default:"10" description:"Interval between data scrapes"`
}

// Execute scraper program
func Execute() {
	klogFs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(klogFs)
	conf := &Config{}
	p := flags.NewParser(conf, flags.Default|flags.IgnoreUnknown)
	fs, err := p.Parse()
	if err != nil {
		if _, ok := err.(*flags.Error); !ok {
			klog.Exitf("Error parsing args: %v", err)
		}
		return
	}
	klogFs.Parse(fs)

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
		klog.Exit(err)
	}
	klog.V(1).Infof("PDU Info: %+v", res)

	if err := scraper.Run(q, conf.Interval); err != nil {
		klog.Exit(err)
	}
}

func metrics(c Config) {
	if !c.Metrics {
		return
	}

	klog.V(1).Infof("Starting Prometheus metrics server on %d", c.Port)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil); err != nil {
		klog.Errorf("HTTP server error: %v", err)
	}
}
