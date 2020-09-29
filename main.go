// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"influx_proxy/backend"
	"influx_proxy/service"
)

var (
	ConfigFile  string
	LogFilePath string
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	flag.StringVar(&LogFilePath, "log-file-path", "", "Log output file.")
	flag.StringVar(&ConfigFile, "config", "config.json", "Configuration file.")
	flag.Parse()
}

func initLog() {
	if LogFilePath == "" {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(&lumberjack.Logger{
			Filename:   LogFilePath,
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     7,
		})
	}
}

func main() {
	initLog()
	if ConfigFile == "" {
		log.Printf("Cannot find configuration file.")
		os.Exit(1)

	}
	cfg, err := backend.LoadConfigFile(ConfigFile)
	if err != nil {
		log.Print("Load config failed: ", err)
		return
	}
	log.Printf("Config file loaded.")
	proxyConfig := cfg.Proxy
	// Build InfluxCluster
	cluster := backend.NewInfluxCluster(cfg)
	err = cluster.Init()
	if err != nil {
		log.Printf("Load influx-db cluster configuration failed: %s", err)
		return
	}

	mux := http.NewServeMux()
	service.NewHttpService(cluster, proxyConfig.DB).Register(mux)
	server := &http.Server{
		Addr:        proxyConfig.ListenAddr,
		Handler:     mux,
		IdleTimeout: time.Duration(proxyConfig.IdleTimeout) * time.Second,
	}
	if proxyConfig.IdleTimeout <= 0 {
		server.IdleTimeout = 10 * time.Second
	}
	err = server.ListenAndServe()
	if err != nil {
		log.Print(err)
		return
	}
	log.Printf("Proxy service started.")
}
