# InfluxDB Proxy

This project adds a basic high availability layer to InfluxDB.

NOTE: influx-proxy must be built with Go 1.11+, don't implement udp.

## Why


We used [InfluxDB Relay](https://github.com/influxdata/influxdb-relay) before, but it doesn't support some demands. 
We use grafana for visualizing time series data, so we need add datasource for grafana. We need change the datasource config when influxdb is down. 
We need transfer data across idc, but Relay doesn't support gzip. 
It's inconvenient to analyse data with connecting different influxdb.
Therefore, we made InfluxDB Proxy. 

## Features

* Support gzip.
* Support query.
* Filter some dangerous influxql.
* Transparent for clients.
* Cache data to file when write failed, then rewrite.

## Requirements

* Golang >= 1.11
* Redis-server
* Python >= 2.7

## Usage

```sh
$ # install redis-server
$ yum install redis
$ # start redis-server on 6379 port
$ redis-server --port 6379 &
$ # Install influxdb-proxy to ./bin
$ cd $PATH_TO_PROXY
$ make
$ # Edit config.py and execute it
$ python config.py
$ # Start influx-proxy!
$ ./bin/influxdb-proxy -redis localhost:6379 [--redis-pwd xxx --redis-db 0]
```

## Configuration

Example configuration file is at [config.py](config.py). 
We use `config.py` to generate configurations to Redis.

## Description

The architecture is fairly simple, one InfluxDB Proxy process and two or more InfluxDB processes. The Proxy should point HTTP requests with measurements to the two InfluxDB servers.

The setup should look like this:

```
        ┌─────────────────┐
        │writes & queries │
        └─────────────────┘
                 │
                 ▼
         ┌───────────────┐
         │               │
         │InfluxDB Proxy │
         |  (only http)  |
         │               │         
         └───────────────┘       
                 │
                 ▼
        ┌─────────────────┐
        │   measurements  │
        └─────────────────┘
          |              |       
        ┌─┼──────────────┘       
        │ └──────────────┐       
        ▼                ▼       
  ┌──────────┐      ┌──────────┐  
  │          │      │          │  
  │ InfluxDB │      │ InfluxDB │
  │          │      │          │
  └──────────┘      └──────────┘
```

Measurements match principle:

* Exact-match first. For instance, we use `cpu.load` for measurement's name. The KEYMAPS has `cpu` and `cpu.load` keys.
It will use the `cpu.load` corresponding backends.

* I removed prefix-match because it will make administrator confused. (For instance, we use `cpu.load` for measurement's name. The KEYMAPS  only has `cpu` key.
It will use the `cpu` corresponding backends.)

## Query Commands


### Unsupported commands

* `DELETE` 
* `DROP`  
* `GRANT`
* `REVOKE`
* `SHOW.*MEASUREMENTS`
* Cross measurements queries

### Supported commands

Only support match the following commands.

* Most of `SELECT` clauses (only 1 measurement)
* Most of `SHOW.*FROM` clauses (only 1 measurement)

## Improvements from the Original Proxy

1. Remove `KEYS` usage when configuring the cluster.
2. Remove prefix-match on measurements to avoid confusion.
3. Support non-measurement-crossing queries using `SELECT` clause.
4. Integrated with `go mod`.
5. Add an HTTP API `/meta` to get the metadata of the cluster, 
including InfluxDB data nodes, proxies.

## TODO

1. Support batch-write protocol to improve throughput.
2. Support more query functions.
3. Support multi-thread message forwarding, with more connections.
4. Integrated with etcd (or other distributed system management service) to build a stronger sentinel.

## License

MIT. 




