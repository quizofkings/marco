package main

import (
	"time"

	"encoding/json"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"

	log "github.com/sirupsen/logrus"
)

type VttabletData struct {
	Alias      VttabletDataAlias `json:"alias"`
	Ip         string            `json:"ip"`       // ip used for stats
	TabletPort VttabletDataPort  `json:"port_map"` //port used for stats
	Keyspace   string            `json:"keyspace"`
	Shard      string            `json:"shard"`
}

type VttabletDataAlias struct {
	Cell string `json:"cell"`
	Uid  int    `json:"uid"`
}

type VttabletDataPort struct {
	Port int `json:"vt"`
}

func discoverLocalTopologyServers(globalETCDEndpoints []string, cells []string) (etcdLocalEndpoints []string) {

	log.Info("Connecting to global etcd endpoint")

	c, err := newEtcd(globalETCDEndpoints)
	if err != nil {
		log.Info("Error Connecting to global etcd server: ", err)
	}
	kapi := etcd.NewKeysAPI(c)

	for _, cell := range cells {
		resp, err := kapi.Get(context.Background(), "/vt/cells/"+cell, nil)
		if err != nil {
			log.Info("Error in global etcd discovery: ", err)
		} else {
			etcdLocalEndpoints = append(etcdLocalEndpoints, resp.Node.Value)
		}
	}

	return
}

func fetchTabletData(localEtcdEndpoints []string) (result []VttabletData) {

	for _, endpoint := range localEtcdEndpoints {
		c, err := newEtcd([]string{endpoint})
		kapi := etcd.NewKeysAPI(c)

		log.Info("Connecting to local etcd endpoint at ", endpoint)
		resp, err := kapi.Get(context.Background(), "/vt/tablets", nil)
		if err != nil {
			log.Info("failed to read /vt/tablets from ", endpoint)
		} else {
			tablets := resp.Node.Nodes
			for _, tablet := range tablets {
				res, err := kapi.Get(context.Background(), tablet.Key+"/_Data", nil)
				if err != nil {
					log.Info("failed to read /_Data from ", tablet.Key)
				} else {
					var tmp VttabletData
					json.Unmarshal([]byte(res.Node.Value), &tmp)
					if err != nil {
						log.Info("failed to unmarshal :", err)
					}
					result = append(result, tmp)
				}
			}
		}
	}
	return
}

func newEtcd(endpoints []string) (client etcd.Client, err error) {
	cfg := etcd.Config{
		Endpoints:               endpoints,
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	client, err = etcd.New(cfg)
	return
}

func DiscoverVttabletsData(etcdEndpoint string, cells []string) []VttabletData {
	endpoints := discoverLocalTopologyServers([]string{etcdEndpoint}, cells) //TODO need to fetch these parameters from config
	return fetchTabletData(endpoints)
}
