package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

const httpTimeout = 2 * time.Second

func FetchData(ip string, port int) (err error, body []byte) {
	port_num := strconv.Itoa(port)
	timeout := time.Duration(httpTimeout)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get("http://" + ip + ":" + port_num + "/debug/vars")
	if err != nil {
		log.Info("timeout from server:", ip, ":", port_num)
		return err, nil
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info("Error in Reading body")
	}
	return nil, body
}

func (stat *VttabletStats) ExportData(data VttabletData) (err error) {
	err, body := FetchData(data.Ip, data.TabletPort.Port)
	if err != nil {
		return err
	}
	json.Unmarshal(body, stat)
	return nil
}

func ExportVtgateData(IP string, port int) map[string]interface{} {
	err, body := FetchData(IP, port)
	if err != nil {
		log.Info("ExportData could not get body") //Debug
		return nil
	}

	var tmp map[string]interface{}
	if err = json.Unmarshal(body, &tmp); err != nil {
		return nil
	}
	res := tmp["VttabletCall"].(map[string]interface{})
	return res
}
