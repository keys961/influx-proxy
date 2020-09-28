#!/usr/bin/python
# -*- coding: utf-8 -*-
'''
@date: 2017-01-24
@author: Shell.Xu
@copyright: 2017, Eleme <zhixiang.xu@ele.me>
@license: MIT
'''
from __future__ import absolute_import, division, \
    print_function, unicode_literals
import sys
import getopt
import json
import redis


# backends key use for KEYMAPS, NODES, cache file
# url: influxdb addr or other http backend which supports influxdb line protocol
# db: influxdb db
# zone: same zone first query
# interval: default config is 1000ms, wait 1 second write whether point count has bigger than maxrowlimit config
# timeout: default config is 10000ms, write timeout until 10 seconds
# timeoutquery: default config is 600000ms, query timeout until 600 seconds
# maxrowlimit: default config is 10000, wait 10000 points write
# checkinterval: default config is 1000ms, check backend active every 1 second
# rewriteinterval: default config is 10000ms, rewrite every 10 seconds
# writeonly: default 0
BACKENDS = {
    'node1': {
        'url': 'http://10.100.2.180:8086',
        'db': 'citibike',
        'zone': 'local',
        'interval': 1000,
        'timeout': 10000,
        'timeoutQuery': 600000,
        'maxRowLimit': 10000,
        'checkInterval': 1000,
        'rewriteInterval': 10000,
    },
    'node2': {
        'url': 'http://10.100.2.190:8086',
        'db': 'citibike',
        'zone':'local',
        'interval': 1000,
        'timeout': 10000,
        'timeoutQuery': 600000,
        'maxRowLimit': 10000,
        'checkInterval': 1000,
        'rewriteInterval': 10000,
    },
}

# measurement:[backends keys], the key must be in the BACKENDS
# data with the measurement will write to the backends
KEYMAPS = {
    'station_data1': ['node1'],
    'station_data2': ['node2'],
    'station_data': ['node1', 'node2'],
    'cpu': ['node1'],
    'temperature': ['node2'],
    '_default_': ['node1', 'node2']
}

# this config will cover default_node config
# listenaddr: proxy listen addr
# db: proxy db, client's db must be same with it
# zone: use for query
# nexts: the backends keys, will accept all data, split with ','
# interval: collect Statistics
# idletimeout: keep-alives wait time
# writetracing: enable logging for the write,default is 0
# querytracing: enable logging for the query,default is 0
PROXIES = {
    'p1': {
        'listenAddr': ':8087',
        'db': 'citibike',
        'zone': 'local',
        'interval': 10,
        'idleTimeout': 10,
        'writeTracing': 0,
        'queryTracing': 0,
    },
    'p2': {
        'listenAddr': ':8087',
        'db': 'citibike',
        'zone': 'local',
        'interval': 10,
        'idleTimeout': 10,
        'writeTracing': 0,
        'queryTracing': 0,
    }
}


def cleanups(client, keys):
    for key in keys:
        client.delete(key)


def write_configs(client, o, outer_key):
    client.delete(outer_key)
    for k, v in o.items():
        client.hset(outer_key, k, json.dumps(v))


def write_config(client, d, name):
    for k, v in d.items():
        client.hset(name, k, v)


def main():
    optlist, args = getopt.getopt(sys.argv[1:], 'd:hH:p:P:')
    optdict = dict(optlist)

    if '-h' in optdict:
        print(main.__doc__)
        return

    client = redis.StrictRedis(
        host=optdict.get('-H', 'localhost'),
        port=int(optdict.get('-p', '6379')),
        db=int(optdict.get('-d', '0')),
        password=optdict.get('-P', '')
    )

    cleanups(client, ['b:', 'm:', 'n:'])

    write_configs(client, BACKENDS, 'b:')
    write_configs(client, PROXIES, 'n:')
    write_configs(client, KEYMAPS, 'm:')


if __name__ == '__main__':
    main()
