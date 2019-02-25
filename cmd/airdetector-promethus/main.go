package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eternal-flame-AD/phicomm-airdetector"
)

var prefix string
var metricAddr string

var metrics = make(map[[6]byte]readingWithTimestamp)
var metricsLock = sync.RWMutex{}
var metricHeadTpl = template.Must(template.New("metric_head").Parse(strings.TrimSpace(`
# HELP {{.Prefix}}humidity humidity in %H
# TYPE {{.Prefix}}humidity gauge
# HELP {{.Prefix}}temperature temperature in degC
# TYPE {{.Prefix}}temperature gauge
# HELP {{.Prefix}}pm25 pm25 in ug/m3
# TYPE {{.Prefix}}pm25 gauge
# HELP {{.Prefix}}hcho hcho in mg/m3
# TYPE {{.Prefix}}hcho gauge
`)))
var singlemetricTpl = template.Must(template.New("metric").Parse(strings.TrimSpace(`
{{.Prefix}}humidity{device={{.MAC}}} {{.R.Reading.Humidity}} {{.R.Timestamp}}
{{.Prefix}}temperature{device={{.MAC}}}  {{.R.Reading.Temperature}} {{.R.Timestamp}}
{{.Prefix}}pm25{device={{.MAC}}}  {{.R.Reading.PM25}} {{.R.Timestamp}}
{{.Prefix}}hcho{device={{.MAC}}}  {{.R.Reading.HCHO}} {{.R.Timestamp}}
`)))

type readingWithTimestamp struct {
	Reading   airdetector.Reading
	Timestamp int64
}

func init() {
	p := flag.String("p", "airdetector", "metric prefix")
	m := flag.String("m", ":9100", "metrics address")
	flag.Parse()

	prefix = *p
	metricAddr = *m
}

func recordReading(meas airdetector.ReadingWithConnInfo) {
	rec := readingWithTimestamp{
		Reading:   meas.Reading,
		Timestamp: time.Now().Unix() * 1000,
	}
	metricsLock.Lock()
	defer metricsLock.Unlock()
	metrics[meas.DeviceMAC] = rec
}

func main() {

	go func() {
		measurements, err := airdetector.Listen()
		if err != nil {
			panic(err)
		}
		for meas := range measurements {
			fmt.Printf("%s: device %x => PM25:%d HCHO:%.2f T:%.1f H:%.1f\n", time.Now().Format("2006-01-02T15:04:05-0700"), meas.DeviceMAC, meas.PM25, meas.HCHO, meas.Temperature, meas.Humidity)
			recordReading(meas)
		}
	}()

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/plain; version=0.0.4")
		metricsLock.RLock()
		defer metricsLock.RUnlock()
		metricPrefix := ""
		if prefix != "" {
			metricPrefix = prefix + "_"
		}
		metricHeadTpl.Execute(w, struct {
			Prefix string
		}{metricPrefix})
		w.Write([]byte("\n"))
		for mac, reading := range metrics {
			macStr := fmt.Sprintf("%x", mac)
			singlemetricTpl.Execute(w, struct {
				MAC    string
				Prefix string
				R      readingWithTimestamp
			}{macStr, metricPrefix, reading})
			w.Write([]byte("\n"))
		}
	})

	log.Println("promethus metrics running at " + metricAddr)
	log.Fatal(http.ListenAndServe(metricAddr, nil))
}
