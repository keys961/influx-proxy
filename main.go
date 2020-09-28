// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/redis.v5"

	"influx_proxy/backend"
	"influx_proxy/service"
)

var (
	ConfigFile  string
	NodeName    string
	RedisAddr   string
	RedisPwd    string
	RedisDb     int
	LogFilePath string
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	flag.StringVar(&LogFilePath, "log-file-path", "", "Log output file.")
	flag.StringVar(&ConfigFile, "config", "", "Configuration file.")
	flag.StringVar(&NodeName, "node", "p1", "Node name")
	flag.StringVar(&RedisAddr, "redis", "localhost:6379", "Redis address")
	flag.StringVar(&RedisPwd, "redis-pwd", "", "Redis password")
	flag.IntVar(&RedisDb, "redis-db", 0, "Redis db id")
	flag.Parse()
}

type ProxyConfig struct {
	redis.Options
	Node string
}

func LoadJson(cfgFile string, cfg interface{}) (err error) {
	file, err := os.Open(cfgFile)
	if err != nil {
		return
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(&cfg)
	return
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

	var err error
	var cfg ProxyConfig

	if ConfigFile != "" {
		err = LoadJson(ConfigFile, &cfg)
		if err != nil {
			log.Print("Load config failed: ", err)
			return
		}
		log.Printf("Config file loaded.")
	}
	if NodeName != "" {
		cfg.Node = NodeName
	}
	if RedisAddr != "" {
		cfg.Addr = RedisAddr
		cfg.Password = RedisPwd
		cfg.DB = RedisDb
	}

	rcs := backend.NewRedisConfigSource(&cfg.Options, cfg.Node)
	// Fetch node cluster info from redis
	proxyConfig, err := rcs.GetProxyConfig()
	if err != nil {
		log.Printf("Config source load failed.")
		return
	}
	// Build InfluxCluster
	ic := backend.NewInfluxCluster(rcs, &proxyConfig)
	err = ic.LoadConfig()
	if err != nil {
		log.Printf("Load influx-db cluster configuration failed: %s", err)
		return
	}

	mux := http.NewServeMux()
	service.NewHttpService(ic, proxyConfig.DB).Register(mux)
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
