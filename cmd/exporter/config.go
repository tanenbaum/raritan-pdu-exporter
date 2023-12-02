package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
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
	Timeout        int             `json:"timeout" yaml:"timeout"`
	Username       string          `json:"username" yaml:"username"`
	Password       string          `json:"password" yaml:"password"`
	Metrics        bool            `json:"metrics" yaml:"metrics"`
	Port           uint            `json:"port" yaml:"port"`
	Interval       uint            `json:"interval" yaml:"interval"`
	ExporterLabels map[string]bool `json:"exporter_labels" yaml:"exporter_labels"`
	// struct {
	// 	UseConfigName   *bool `json:"use_config_name" yaml:"use_config_name"`
	// 	SerialNumber    *bool `json:"serial_number" yaml:"serial_number"`
	// 	SNMPSysContact  *bool `json:"snmp_sys_contact" yaml:"snmp_sys_contact"`
	// 	SNMPSysName     *bool `json:"snmp_sys_name" yaml:"snmp_sys_name"`
	// 	SNMPSydLocation *bool `json:"snmp_sys_location" yaml:"snmp_sys_location"`
	// }
	PduConfig []PduConfig `json:"pdu_config" yaml:"pdu_config"`
}

type Config struct {
	Metrics        bool            `json:"metrics" yaml:"metrics"`
	Port           uint            `json:"port" yaml:"port"`
	Interval       uint            `json:"interval" yaml:"interval"`
	ExporterLabels map[string]bool `json:"exporter_labels" yaml:"exporter_labels"`
	// struct {
	// 	UseConfigName   *bool `json:"use_config_name" yaml:"use_config_name"`
	// 	SerialNumber    *bool `json:"serial_number" yaml:"serial_number"`
	// 	SNMPSysContact  *bool `json:"snmp_sys_contact" yaml:"snmp_sys_contact"`
	// 	SNMPSysName     *bool `json:"snmp_sys_name" yaml:"snmp_sys_name"`
	// 	SNMPSydLocation *bool `json:"snmp_sys_location" yaml:"snmp_sys_location"`
	// }
	PduConfig []PduConfig `json:"pdu_config" yaml:"pdu_config"`
}

type PduConfig struct {
	Name     string `json:"name" yaml:"name"`
	Address  string `json:"address" yaml:"address"`
	Timeout  int    `json:"timeout" yaml:"timeout"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

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
		ExporterLabels: map[string]bool{
			"use_config_name":   false,
			"serial_number":     true,
			"snmp_sys_contact":  false,
			"snmp_sys_name":     false,
			"snmp_sys_location": false,
		},
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

		for k, v := range fileConfig.ExporterLabels {
			conf.ExporterLabels[k] = v
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

func LoadConfig(args []string) (*Config, error) {
	klogFs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(klogFs)
	cliConf := &CliConfig{}

	p := flags.NewParser(cliConf, flags.Default|flags.IgnoreUnknown)

	fs, err := p.ParseArgs(args)
	if err != nil {
		if _, ok := err.(*flags.Error); !ok {
			return nil, fmt.Errorf("error parsing args: %v", err)
		}
		return nil, err
	}

	klogFs.Parse(fs)

	conf, err := cliConf.GetConfig()
	if err != nil {
		return nil, err
	}
	klog.Infof("Server config: port=%d metrics=%t\n", conf.Port, conf.Metrics)
	for _, p := range conf.PduConfig {
		var name string
		if p.Name != "" {
			name = p.Name
		} else {
			name = "<no name defined>"
		}
		klog.Infof("PDU config: name=%s url=%s\n", name, p.Url())
	}

	return conf, nil
}
