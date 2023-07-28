package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/goji/httpauth"
	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"github.com/tanenbaum/raritan-pdu-exporter/internal/rpc"
	"k8s.io/klog"
)

// Config for stub
type Config struct {
	Username string `short:"u" long:"username" env:"PDU_USERNAME" description:"Username for server basic auth"`
	Password string `short:"p" long:"password" env:"PDU_PASSWORD" description:"Password for server basic auth"`
	Port     uint   `long:"port" default:"3000"`
}

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
	_ = klogFs.Parse(fs)

	klog.V(1).Infof("Using config: %+v", conf)

	auth := httpauth.SimpleBasicAuth(conf.Username, conf.Password)
	bulkClient := rpc.NewClient(0, rpc.Auth{
		Username: conf.Username,
		Password: conf.Password,
	})

	r := mux.NewRouter()
	r.HandleFunc("/model/pdu/0", pduHandler)
	r.HandleFunc("/bulk", bulkHandler(bulkClient, conf.Port))
	r.HandleFunc("/model/inlet/{id:[0-9]+}", inletsHandler)
	r.HandleFunc("/model/outlet/{id:[0-9]+}", outletsHandler)
	r.HandleFunc("/tfwopaque/{type}/{id:[0-9]+}", ocpHandler)
	r.HandleFunc("/tfwopaque/{id:[0-9]+}/{sensor}", sensorHandler)
	r.HandleFunc("/model/{type}/{id:[0-9]+}/{sensor}", sensorHandler)

	klog.Exit(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), logger(auth(r))))
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		klog.V(2).Infof("HTTP Request: URL: %s, method: %s, remote: %s", r.URL.String(), r.Method, r.RemoteAddr)
		next.ServeHTTP(w, r)
		klog.V(2).Infof("HTTP Response: Headers: %v", w.Header())
	})
}
