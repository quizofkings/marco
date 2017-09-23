package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
)

var (
	globalEtcdEndpoint string
	vtgateIP           string
	vtgatePort         int
	cells              []string
	httpPort           int
)

// TODO: ADD required stats
type VttabletStats struct {
	ConnAccepted int           `json:"ConnAccepted"`
	QPS          VttabletQuery `json:"QPS"`
}

type VttabletCall struct {
	TotalCount string `json:"TotalCount"`
	TotalTime  string `json:"TotalTime"`
}

type VttabletQuery struct {
	All []float32 `json:"All"`
}

type VtgateStats struct {
	TotalCount float64
	TotalTime  float64
	Histograms []VtgateHistogram
}

type VtgateHistogram struct {
	Keyspace          string
	Shard             string
	Bucket500000      float64
	Bucket1000000     float64
	Bucket5000000     float64
	Bucket10000000    float64
	Bucket50000000    float64
	Bucket100000000   float64
	Bucket500000000   float64
	Bucket1000000000  float64
	Bucket5000000000  float64
	Bucket10000000000 float64
	Inf               float64
	Count             float64
	Time              float64
}

type prometheusData struct {
	Data  VttabletData
	Stats VttabletStats
} //pointers ???

func init() {

	var vtgateAddress string
	var tmpCells string

	flag.StringVar(&globalEtcdEndpoint, "etcd", "127.0.0.1:2379", "global etcd endpoint")
	flag.StringVar(&vtgateAddress, "vtgate", "127.0.0.1:5001", "vtgate address")
	flag.StringVar(&tmpCells, "cells", "vitess", "comma seperated cell names to monitor")
	flag.IntVar(&httpPort, "port", 80, "http port to expose statistics")

	flag.Parse()

	if strings.HasPrefix(strings.ToLower(vtgateAddress), "http://") {
		vtgateAddress = strings.TrimPrefix(vtgateAddress, "http://")
	}
	vttmp := strings.Split(vtgateAddress, ":")
	if len(vttmp) != 2 {
		log.Info("--vtgate usage: IP:port_number")
		os.Exit(0)
	} else {
		vtgateIP = vttmp[0]
		i, err := strconv.Atoi(vttmp[1])
		if err != nil {
			log.Info("port number needs to be a digit")
			os.Exit(0)
		} else {
			vtgatePort = i
		}
	}

	if !strings.HasPrefix(strings.ToLower(globalEtcdEndpoint), "http://") {
		globalEtcdEndpoint = "http://" + globalEtcdEndpoint
	}
	if len(strings.Split(globalEtcdEndpoint, ":")) != 3 {
		log.Info("--etcd usage: IP:port_number")
		os.Exit(0)
	}

	cells = strings.Split(tmpCells, ",")

}

func vtgateHandler(w http.ResponseWriter, r *http.Request) {

	var stats VtgateStats
	var histograms []VtgateHistogram

	data := ExportVtgateData(vtgateIP, vtgatePort)
	if data == nil {
		log.Info("could not get vtgate data")
	} else {

		stats.TotalCount = (data["TotalCount"].(float64))
		stats.TotalTime = data["TotalTime"].(float64) / 10e9
		histogramMaps := data["Histograms"].(map[string]interface{})
		for k, v := range histogramMaps {
			var histogram VtgateHistogram
			nameSplit := strings.Split(k, ".")
			histogram.Keyspace = nameSplit[1]
			histogram.Shard = nameSplit[2]

			tmp := v.(map[string]interface{})
			histogram.Count = tmp["Count"].(float64)
			histogram.Time = tmp["Time"].(float64)
			histogram.Inf = tmp["inf"].(float64)
			histogram.Bucket500000 = tmp["500000"].(float64)
			histogram.Bucket1000000 = tmp["1000000"].(float64)
			histogram.Bucket5000000 = tmp["5000000"].(float64)
			histogram.Bucket10000000 = tmp["10000000"].(float64)
			histogram.Bucket50000000 = tmp["5000000"].(float64)
			histogram.Bucket100000000 = tmp["100000000"].(float64)
			histogram.Bucket500000000 = tmp["500000000"].(float64)
			histogram.Bucket1000000000 = tmp["1000000000"].(float64)
			histogram.Bucket5000000000 = tmp["5000000000"].(float64)
			histogram.Bucket10000000000 = tmp["5000000000"].(float64)

			histograms = append(histograms, histogram)
			//			log.Info(histogram)
		}
		stats.Histograms = histograms

		t, err := template.ParseFiles("vtgate.tmpl")
		if err != nil {
			panic(err)
		} else {
			err = t.Execute(w, &stats)
			if err != nil {
				panic(err)
			}
		}
	}
}

func vttabletHandler(w http.ResponseWriter, r *http.Request) {
	//TODO QPS int series fix

	var promData []prometheusData
	var wg sync.WaitGroup

	tabletsData := DiscoverVttabletsData(globalEtcdEndpoint, cells)

	for _, tablet := range tabletsData {
		wg.Add(1)
		go exportPromData(&promData, tablet, &wg)
	}
	//	log.Info("waiting for tablets to finish ")
	wg.Wait()
	log.Info("Finished Gathering tablet data")

	t, err := template.ParseFiles("vttablet.tmpl")
	if err != nil {
		panic(err)
	} else {
		err = t.Execute(w, &promData)
		if err != nil {
			panic(err)
		}
	}
}

func exportPromData(promData *[]prometheusData, tablet VttabletData, wg *sync.WaitGroup) {
	var stat VttabletStats
	err := stat.ExportData(tablet)
	if err != nil {
		log.Info("Failed to fetch data for tablet ", tablet.Alias.Uid)
	} else {
		log.Info("Tablet data successfully fetched for ", tablet.Alias.Uid) //Debug
		*promData = append(*promData, prometheusData{tablet, stat})
	}
	wg.Done()
}

func main() {
	http.HandleFunc("/vtgate", vtgateHandler)
	http.HandleFunc("/vttablet", vttabletHandler)
	http.ListenAndServe(":"+strconv.Itoa(httpPort), nil)

}
