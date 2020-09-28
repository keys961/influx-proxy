// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package backend

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"influx_proxy/monitor"
)

var (
	ErrBackendNotExist = errors.New("use a backend not exists")
	ErrQueryForbidden  = errors.New("query forbidden")
)

func ScanKey(pointBuf []byte) (key string, err error) {
	var keyBuf [100]byte
	keySlice := keyBuf[0:0]
	bufLen := len(pointBuf)
	for i := 0; i < bufLen; i++ {
		c := pointBuf[i]
		switch c {
		case '\\':
			i++
			keySlice = append(keySlice, pointBuf[i])
		case ' ', ',':
			key = string(keySlice)
			return
		default:
			keySlice = append(keySlice, c)
		}
	}
	return "", io.EOF
}

type InfluxCluster struct {
	lock                  sync.RWMutex
	Zone                  string
	queryExecutor         Queryable
	ForbiddenQuery        []*regexp.Regexp
	ObligatedQuery        []*regexp.Regexp
	redisConfigSrc        *RedisConfigSource
	backends              map[string]BackendApi   // backendName to backend
	measurementToBackends map[string][]BackendApi // measurements to backends
	stats                 *Statistics
	counter               *Statistics
	ticker                *time.Ticker
	defaultTags           map[string]string
	WriteTracing          int
	QueryTracing          int
}

type Statistics struct {
	QueryRequests        int64
	QueryRequestsFail    int64
	WriteRequests        int64
	WriteRequestsFail    int64
	PingRequests         int64
	PingRequestsFail     int64
	PointsWritten        int64
	PointsWrittenFail    int64
	WriteRequestDuration int64
	QueryRequestDuration int64
}

type ClusterMetadata struct {
	Proxies               map[string]*ProxyConfig   `json:"proxies"`
	Backends              map[string]*BackendConfig `json:"backends"`
	BackendStatus         map[string]bool           `json:"backendStatus"`
	MeasurementToBackends map[string][]string       `json:"measurementToBackends"`
}

func NewInfluxCluster(redisConfigSrc *RedisConfigSource, nodeConfig *ProxyConfig) (ic *InfluxCluster) {
	ic = &InfluxCluster{
		Zone:           nodeConfig.Zone,
		queryExecutor:  &InfluxQLExecutor{},
		redisConfigSrc: redisConfigSrc,
		stats:          &Statistics{},
		counter:        &Statistics{},
		ticker:         time.NewTicker(10 * time.Second),
		defaultTags:    map[string]string{"addr": nodeConfig.ListenAddr},
		WriteTracing:   nodeConfig.WriteTracing,
		QueryTracing:   nodeConfig.QueryTracing,
	}
	host, err := os.Hostname()
	if err != nil {
		log.Println(err)
	}
	ic.defaultTags["host"] = host
	if nodeConfig.Interval > 0 {
		ic.ticker = time.NewTicker(time.Second * time.Duration(nodeConfig.Interval))
	}

	err = ic.ForbidQuery(ForbidCommands)
	if err != nil {
		panic(err)
		return
	}
	err = ic.EnsureQuery(SupportCommands)
	if err != nil {
		panic(err)
		return
	}

	// feature
	go ic.statistics()
	return
}

func (ic *InfluxCluster) statistics() {
	// how to quit
	for {
		<-ic.ticker.C
		ic.Flush()
		ic.counter = (*Statistics)(atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&ic.stats)),
			unsafe.Pointer(ic.counter)))
		err := ic.WriteStatistics()
		if err != nil {
			log.Println(err)
		}
	}
}

func (ic *InfluxCluster) Flush() {
	ic.counter.QueryRequests = 0
	ic.counter.QueryRequestsFail = 0
	ic.counter.WriteRequests = 0
	ic.counter.WriteRequestsFail = 0
	ic.counter.PingRequests = 0
	ic.counter.PingRequestsFail = 0
	ic.counter.PointsWritten = 0
	ic.counter.PointsWrittenFail = 0
	ic.counter.WriteRequestDuration = 0
	ic.counter.QueryRequestDuration = 0
}

func (ic *InfluxCluster) WriteStatistics() (err error) {
	metric := &monitor.Metric{
		Name: "influxdb.cluster",
		Tags: ic.defaultTags,
		Fields: map[string]interface{}{
			"statQueryRequest":         ic.counter.QueryRequests,
			"statQueryRequestFail":     ic.counter.QueryRequestsFail,
			"statWriteRequest":         ic.counter.WriteRequests,
			"statWriteRequestFail":     ic.counter.WriteRequestsFail,
			"statPingRequest":          ic.counter.PingRequests,
			"statPingRequestFail":      ic.counter.PingRequestsFail,
			"statPointsWritten":        ic.counter.PointsWritten,
			"statPointsWrittenFail":    ic.counter.PointsWrittenFail,
			"statQueryRequestDuration": ic.counter.QueryRequestDuration,
			"statWriteRequestDuration": ic.counter.WriteRequestDuration,
		},
		Time: time.Now(),
	}
	line, err := metric.ParseToLine()
	if err != nil {
		return
	}
	return ic.Write([]byte(line + "\n"))
}

func (ic *InfluxCluster) ForbidQuery(s string) (err error) {
	r, err := regexp.Compile(s)
	if err != nil {
		return
	}

	ic.lock.Lock()
	defer ic.lock.Unlock()
	ic.ForbiddenQuery = append(ic.ForbiddenQuery, r)
	return
}

func (ic *InfluxCluster) EnsureQuery(s string) (err error) {
	r, err := regexp.Compile(s)
	if err != nil {
		return
	}

	ic.lock.Lock()
	defer ic.lock.Unlock()
	ic.ObligatedQuery = append(ic.ObligatedQuery, r)
	return
}

func (ic *InfluxCluster) GetClusterMetadata() (metadata *ClusterMetadata, err error) {
	metadata = &ClusterMetadata{}
	metadata.Backends, err = ic.redisConfigSrc.GetBackendsConfig()
	if err != nil {
		log.Printf("Error fetching backend configuration\n")
		return
	}
	metadata.BackendStatus = make(map[string]bool)
	for backendName, _ := range metadata.Backends {
		metadata.BackendStatus[backendName] = ic.backends[backendName].IsActive()
	}
	metadata.MeasurementToBackends, err = ic.redisConfigSrc.GetMeasurementsConfig()
	if err != nil {
		log.Printf("Error fetching measurement configuration\n")
		return
	}
	metadata.Proxies, err = ic.redisConfigSrc.GetProxiesConfig()
	if err != nil {
		log.Printf("Error fetching proxy configuration\n")
		return
	}
	return
}

func (ic *InfluxCluster) loadBackends() (backends map[string]BackendApi, err error) {
	backends = make(map[string]BackendApi)

	backendConfigs, err := ic.redisConfigSrc.GetBackendsConfig()
	if err != nil {
		return
	}

	for name, cfg := range backendConfigs {
		backends[name], err = NewBackend(cfg, name)
		if err != nil {
			log.Printf("Create backend error: %s", err)
			return
		}
	}
	return
}

func (ic *InfluxCluster) loadMeasurements(backends map[string]BackendApi) (measurementToBackends map[string][]BackendApi, err error) {
	measurementToBackends = make(map[string][]BackendApi)

	measurementMap, err := ic.redisConfigSrc.GetMeasurementsConfig()
	if err != nil {
		return
	}

	for measurementName, backendNames := range measurementMap {
		var backendList []BackendApi
		for _, backendName := range backendNames {
			backend, ok := backends[backendName]
			if !ok {
				err = ErrBackendNotExist
				log.Println(backendName, err)
				continue
			}
			backendList = append(backendList, backend)
		}
		measurementToBackends[measurementName] = backendList
	}
	return
}

func (ic *InfluxCluster) LoadConfig() (err error) {
	backends, err := ic.loadBackends()
	if err != nil {
		return
	}

	measurementToBackends, err := ic.loadMeasurements(backends)
	if err != nil {
		return
	}

	ic.lock.Lock()
	originBackends := ic.backends
	ic.backends = backends
	ic.measurementToBackends = measurementToBackends
	ic.lock.Unlock()
	// Close origin backends
	for name, bs := range originBackends {
		err = bs.Close()
		if err != nil {
			log.Printf("Fail in close backend %s", name)
		}
	}
	return
}

func (ic *InfluxCluster) Ping() (version string, err error) {
	atomic.AddInt64(&ic.stats.PingRequests, 1)
	version = VERSION
	return
}

func (ic *InfluxCluster) CheckQuery(q string) (err error) {
	ic.lock.RLock()
	defer ic.lock.RUnlock()

	if len(ic.ForbiddenQuery) != 0 {
		for _, fq := range ic.ForbiddenQuery {
			if fq.MatchString(q) {
				return ErrQueryForbidden
			}
		}
	}

	if len(ic.ObligatedQuery) != 0 {
		for _, pq := range ic.ObligatedQuery {
			if pq.MatchString(q) {
				return
			}
		}
		return ErrQueryForbidden
	}

	return
}

func (ic *InfluxCluster) GetBackends(key string) (backends []BackendApi, ok bool) {
	ic.lock.RLock()
	defer ic.lock.RUnlock()

	backends, ok = ic.measurementToBackends[key]
	// match use prefix
	if !ok {
		for k, v := range ic.measurementToBackends {
			if strings.HasPrefix(key, k) {
				backends = v
				ok = true
				break
			}
		}
	}

	if !ok {
		backends, ok = ic.measurementToBackends["_default_"]
	}

	return
}

func (ic *InfluxCluster) Query(w http.ResponseWriter, req *http.Request) (err error) {
	atomic.AddInt64(&ic.stats.QueryRequests, 1)
	defer func(start time.Time) {
		atomic.AddInt64(&ic.stats.QueryRequestDuration, time.Since(start).Nanoseconds())
	}(time.Now())

	switch req.Method {
	case "GET", "POST":
	default:
		w.WriteHeader(400)
		_, _ = w.Write([]byte("illegal method"))
		atomic.AddInt64(&ic.stats.QueryRequestsFail, 1)
		return
	}

	// TODO: all query in q?
	q := strings.TrimSpace(req.FormValue("q"))
	if q == "" {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("empty query"))
		atomic.AddInt64(&ic.stats.QueryRequestsFail, 1)
		return
	}

	err = ic.queryExecutor.Query(w, req)
	if err == nil {
		return
	}

	err = ic.CheckQuery(q)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("query forbidden"))
		atomic.AddInt64(&ic.stats.QueryRequestsFail, 1)
		return
	}

	key, err := GetMeasurementFromInfluxQL(q)
	if err != nil {
		log.Printf("can't get measurement: %s\n", q)
		w.WriteHeader(400)
		w.Write([]byte("can't get measurement"))
		atomic.AddInt64(&ic.stats.QueryRequestsFail, 1)
		return
	}
	if len(key) > 1 {
		log.Printf("don't support multiple measurements: %s\n", q)
		w.WriteHeader(400)
		w.Write([]byte("don't support multiple measurements"))
		atomic.AddInt64(&ic.stats.QueryRequestsFail, 1)
		return
	}

	apis, ok := ic.GetBackends(key[0])
	if !ok {
		log.Printf("unknown measurement: %s,the query is %s\n", key, q)
		w.WriteHeader(400)
		w.Write([]byte("unknown measurement"))
		atomic.AddInt64(&ic.stats.QueryRequestsFail, 1)
		return
	}

	// same zone first, other zone. pass non-active.
	// TODO: better way?

	for _, api := range apis {
		if api.GetZone() != ic.Zone {
			continue
		}
		if !api.IsActive() || api.IsWriteOnly() {
			continue
		}
		err = api.Query(w, req)
		if err == nil {
			return
		}
	}

	for _, api := range apis {
		if api.GetZone() == ic.Zone {
			continue
		}
		if !api.IsActive() {
			continue
		}
		err = api.Query(w, req)
		if err == nil {
			return
		}
	}

	w.WriteHeader(400)
	_, _ = w.Write([]byte("query error"))
	atomic.AddInt64(&ic.stats.QueryRequestsFail, 1)
	return
}

// Wrong in one row will not stop others.
// So don't try to return error, just print it.
func (ic *InfluxCluster) WriteRow(line []byte) {
	atomic.AddInt64(&ic.stats.PointsWritten, 1)
	// maybe trim?
	line = bytes.TrimRight(line, " \t\r\n")

	// empty line, ignore it.
	if len(line) == 0 {
		return
	}

	key, err := ScanKey(line)
	if err != nil {
		log.Printf("scan key error: %s\n", err)
		atomic.AddInt64(&ic.stats.PointsWrittenFail, 1)
		return
	}

	bs, ok := ic.GetBackends(key)
	if !ok {
		log.Printf("new measurement: %s\n", key)
		atomic.AddInt64(&ic.stats.PointsWrittenFail, 1)
		// TODO: new measurement?
		return
	}

	// don't block here for a long time, we just have one worker.
	for _, b := range bs {
		err = b.Write(line)
		if err != nil {
			log.Printf("cluster write fail: %s\n", key)
			atomic.AddInt64(&ic.stats.PointsWrittenFail, 1)
			return
		}
	}
	return
}

func (ic *InfluxCluster) Write(p []byte) (err error) {
	atomic.AddInt64(&ic.stats.WriteRequests, 1)
	defer func(start time.Time) {
		atomic.AddInt64(&ic.stats.WriteRequestDuration, time.Since(start).Nanoseconds())
	}(time.Now())

	buf := bytes.NewBuffer(p)

	var line []byte
	for {
		line, err = buf.ReadBytes('\n')
		switch err {
		default:
			log.Printf("error: %s\n", err)
			atomic.AddInt64(&ic.stats.WriteRequestsFail, 1)
			return
		case io.EOF, nil:
			err = nil
		}

		if len(line) == 0 {
			break
		}

		ic.WriteRow(line)
	}
	return
}

func (ic *InfluxCluster) Close() (err error) {
	ic.lock.RLock()
	defer ic.lock.RUnlock()
	for name, bs := range ic.backends {
		err = bs.Close()
		if err != nil {
			log.Printf("fail in close backend %s", name)
		}
	}
	return
}
