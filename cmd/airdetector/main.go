package main

import (
	"fmt"
	"time"

	airdetector "github.com/eternal-flame-AD/phicomm-airdetector"
)

func main() {
	measurements, err := airdetector.Listen()
	if err != nil {
		panic(err)
	}
	for meas := range measurements {
		fmt.Printf("%s: device %x => PM25:%d HCHO:%.2f T:%.1f H:%.1f Stable:%t\n", time.Now().Format("2006-01-02T15:04:05-0700"), meas.DeviceMAC, meas.PM25, meas.HCHO, meas.Temperature, meas.Humidity, meas.IsStable)
	}
}
