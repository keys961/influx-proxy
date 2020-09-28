// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package service

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/pprof"
	"strings"

	"influx_proxy/backend"
)

type HttpService struct {
	db string
	ic *backend.InfluxCluster
}

func NewHttpService(ic *backend.InfluxCluster, db string) (hs *HttpService) {
	hs = &HttpService{
		db: db,
		ic: ic,
	}
	if hs.db != "" {
		log.Print("http database: ", hs.db)
	}
	return
}

func (hs *HttpService) Register(mux *http.ServeMux) {
	mux.HandleFunc("/reload", hs.HandleReload)
	mux.HandleFunc("/ping", hs.HandlePing)
	mux.HandleFunc("/query", hs.HandleQuery)
	mux.HandleFunc("/write", hs.HandleWrite)
	mux.HandleFunc("/meta", hs.HandleClusterMeta)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
}

func (hs *HttpService) HandleClusterMeta(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Add("X-Influxdb-Version", backend.VERSION)

	metadata, err := hs.ic.GetClusterMetadata()
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(200)
	_, _ = w.Write(metadataBytes)
	return
}

func (hs *HttpService) HandleReload(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Add("X-Influxdb-Version", backend.VERSION)

	err := hs.ic.LoadConfig()
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(204)
	return
}

func (hs *HttpService) HandlePing(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	version, err := hs.ic.Ping()
	if err != nil {
		panic("WTF")
		return
	}
	w.Header().Add("X-Influxdb-Version", version)
	w.WriteHeader(200)
	_, _ = w.Write([]byte("Pong"))
	return
}

func (hs *HttpService) HandleQuery(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Add("X-Influxdb-Version", backend.VERSION)

	db := req.FormValue("db")
	if hs.db != "" {
		if db != hs.db {
			w.WriteHeader(404)
			_, _ = w.Write([]byte("database not exist."))
			return
		}
	}

	q := strings.TrimSpace(req.FormValue("q"))
	err := hs.ic.Query(w, req)
	if err != nil {
		log.Printf("query error: %s,the query is %s,the client is %s\n", err, q, req.RemoteAddr)
		return
	}
	if hs.ic.QueryTracing != 0 {
		log.Printf("the query is %s,the client is %s\n", q, req.RemoteAddr)
	}

	return
}

func (hs *HttpService) HandleWrite(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Add("X-Influxdb-Version", backend.VERSION)

	if req.Method != "POST" {
		w.WriteHeader(405)
		_, _ = w.Write([]byte("method not allow."))
		return
	}

	db := req.URL.Query().Get("db")

	if hs.db != "" {
		if db != hs.db {
			w.WriteHeader(404)
			_, _ = w.Write([]byte("database not exist."))
			return
		}
	}

	body := req.Body
	if req.Header.Get("Content-Encoding") == "gzip" {
		b, err := gzip.NewReader(req.Body)
		if err != nil {
			w.WriteHeader(400)
			_, _ = w.Write([]byte("unable to decode gzip body"))
			return
		}
		defer b.Close()
		body = b
	}

	p, err := ioutil.ReadAll(body)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	err = hs.ic.Write(p)
	if err == nil {
		w.WriteHeader(204)
	}
	if hs.ic.WriteTracing != 0 {
		log.Printf("Write body received by handler: %s,the client is %s\n", p, req.RemoteAddr)
	}
	return
}
