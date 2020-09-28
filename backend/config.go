// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package backend

import (
	"encoding/json"
	"log"

	"gopkg.in/redis.v5"
)

const (
	VERSION = "0.9"
)

func decode(data string, o interface{}) (err error) {
	err = json.Unmarshal([]byte(data), &o)
	return
}

// Proxy node configuration
type ProxyConfig struct {
	ListenAddr   string `json:"listenAddr"`
	DB           string `json:"db"`
	Zone         string `json:"zone"`
	Interval     int    `json:"interval"`
	IdleTimeout  int    `json:"idleTimeout"`
	WriteTracing int    `json:"writeTracing"`
	QueryTracing int    `json:"queryTracing"`
}

// Influx-db node configuration
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

// Redis stub for querying configuration
type RedisConfigSource struct {
	client *redis.Client
	node   string
	zone   string
}

func NewRedisConfigSource(options *redis.Options, node string) (rcs *RedisConfigSource) {
	rcs = &RedisConfigSource{
		client: redis.NewClient(options),
		node:   node,
	}
	return
}

func (rcs *RedisConfigSource) GetProxiesConfig() (proxiesConfig map[string]*ProxyConfig, err error) {
	proxiesConfig = make(map[string]*ProxyConfig)
	val, err := rcs.client.HGetAll("n:").Result()
	if err != nil {
		log.Printf("Get proxies config error.")
		return
	}

	for proxyName, proxyConfigJson := range val {
		var proxyConfig = ProxyConfig{}
		err = decode(proxyConfigJson, &proxyConfig)
		if err != nil {
			log.Printf("Get proxies config error.")
		}
		proxiesConfig[proxyName] = &proxyConfig
	}
	return
}

func (rcs *RedisConfigSource) GetProxyConfig() (proxyConfig ProxyConfig, err error) {
	val, err := rcs.client.HGetAll("n:").Result()
	if err != nil {
		log.Printf("Redis load error: b:%s", rcs.node)
		return
	}
	proxyConfigJson := val[rcs.node]
	if proxyConfigJson == "" {
		log.Printf("Get proxy config error: n:%s", rcs.node)
		return
	}
	err = decode(proxyConfigJson, &proxyConfig)
	if err != nil {
		log.Printf("Get proxy config error: n:%s", rcs.node)
		return
	}
	log.Printf("Proxy config loaded.")
	return
}

func (rcs *RedisConfigSource) GetBackendsConfig() (backends map[string]*BackendConfig, err error) {
	backends = make(map[string]*BackendConfig)
	backendConfigs, err := rcs.client.HGetAll("b:").Result()
	if err != nil {
		log.Printf("Get backend config error: %s", err)
		return
	}

	var cfg *BackendConfig
	for name, backendConfig := range backendConfigs {
		cfg, err = rcs.loadConfigFromRedis(backendConfig)
		if err != nil {
			log.Printf("Get backend config error: %s", err)
			return
		}
		backends[name] = cfg
	}
	log.Printf("%d backends config loaded from redis.", len(backends))
	return
}

func (rcs *RedisConfigSource) GetMeasurementsConfig() (measurementMap map[string][]string, err error) {
	measurementMap = make(map[string][]string, 0)

	measurements, err := rcs.client.HGetAll("m:").Result()
	if err != nil {
		log.Printf("read redis error: %s", err)
		return
	}

	for measurementName, backendListJson := range measurements {
		var backendList []string
		err = decode(backendListJson, &backendList)
		if err != nil {
			return
		}
		measurementMap[measurementName] = backendList
	}
	log.Printf("%d measurements loaded from redis.", len(measurementMap))
	return
}

func (rcs *RedisConfigSource) loadConfigFromRedis(configJson string) (cfg *BackendConfig, err error) {
	cfg = &BackendConfig{}
	err = decode(configJson, &cfg)
	if err != nil {
		return
	}

	if cfg.Interval == 0 {
		cfg.Interval = 1000
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10000
	}
	if cfg.TimeoutQuery == 0 {
		cfg.TimeoutQuery = 600000
	}
	if cfg.MaxRowLimit == 0 {
		cfg.MaxRowLimit = 10000
	}
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 1000
	}
	if cfg.RewriteInterval == 0 {
		cfg.RewriteInterval = 10000
	}
	return
}
