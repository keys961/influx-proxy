// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package backend

import (
	"encoding/json"
	"log"
	"os"
)

const (
	VERSION = "0.9.1"
)

// Config Configuration file structure
type Config struct {
	Proxy    ProxyConfig              `json:"proxy"`
	Backends map[string]BackendConfig `json:"backends"`
	Keymaps  map[string][]string      `json:"keymaps"`
}

// ProxyConfig Proxy node configuration
type ProxyConfig struct {
	ListenAddr   string `json:"listenAddr"`
	DB           string `json:"db"`
	Zone         string `json:"zone"`
	Interval     int    `json:"interval"`
	IdleTimeout  int    `json:"idleTimeout"`
	WriteTracing int    `json:"writeTracing"`
	QueryTracing int    `json:"queryTracing"`
}

// BackendConfig InfluxDB node configuration
type BackendConfig struct {
	URL             string `json:"url"`
	DB              string `json:"db"`
	Zone            string `json:"zone"`
	Interval        int    `json:"interval"`
	Timeout         int    `json:"timeout"`
	TimeoutQuery    int    `json:"timeoutQuery"`
	MaxRowLimit     int    `json:"maxRowLimit"`
	CheckInterval   int    `json:"checkInterval"`
	RewriteInterval int    `json:"rewriteInterval"`
	WriteOnly       int    `json:"writeOnly"`
}

func LoadConfigFile(fileName string) (cfg *Config, err error) {
	cfg = &Config{}
	file, err := os.Open(fileName)
	if err != nil {
		log.Panic(err)
		return
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(&cfg)
	if err != nil {
		log.Panic(err)
		return
	}

	for _, backend := range cfg.Backends {
		if backend.Interval == 0 {
			backend.Interval = 1000
		}
		if backend.Timeout == 0 {
			backend.Timeout = 10000
		}
		if backend.TimeoutQuery == 0 {
			backend.TimeoutQuery = 600000
		}
		if backend.MaxRowLimit == 0 {
			backend.MaxRowLimit = 10000
		}
		if backend.CheckInterval == 0 {
			backend.CheckInterval = 1000
		}
		if backend.RewriteInterval == 0 {
			backend.RewriteInterval = 10000
		}
	}
	return
}
