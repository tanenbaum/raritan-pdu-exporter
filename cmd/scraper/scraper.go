package main

import (
	"net/url"
	"time"

	"github.com/jessevdk/go-flags"
	"k8s.io/klog"

	"gitlab.com/edgetic/hw/pdu-sensors/internal/raritan"
	"gitlab.com/edgetic/hw/pdu-sensors/internal/rpc"
)

// Config for scraper
type Config struct {
	Address  string `short:"a" long:"address" required:"true" description:"Address of the PDU JSON RPC endpoint"`
	Timeout  int    `long:"timeout" default:"10" description:"Timeout of PDU RPC requests in seconds"`
	Username string `short:"u" long:"username" env:"PDU_USERNAME" description:"Username for PDU access"`
	Password string `short:"p" long:"password" env:"PDU_PASSWORD" description:"Password for PDU access"`
}

// Execute scraper program
func Execute() {
	conf := &Config{}
	p := flags.NewParser(conf, flags.Default|flags.IgnoreUnknown)
	if _, err := p.Parse(); err != nil {
		if _, ok := err.(*flags.Error); !ok {
			klog.Exitf("Error parsing args: %v", err)
		}
		return
	}

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

	res, err := q.GetPDUMetadata()
	if err != nil {
		klog.Exit(err)
	}
	klog.Infof("PDU Metadata: %v", res)
}
