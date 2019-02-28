# phicomm-airdetector

参照[YinHangCode/homebridge-phicomm-air_detector](https://github.com/YinHangCode/homebridge-phicomm-air_detector)使用golang编写的phicomm airdetector m1数据收集服务器。

## Example

```golang
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
		fmt.Printf("%s: device %x => PM25:%d HCHO:%.2f T:%.1f H:%.1f\n", time.Now().Format("2006-01-02T15:04:05-0700"), meas.DeviceMAC, meas.PM25, meas.HCHO, meas.Temperature, meas.Humidity)
	}
}
```

## Usage

1. 运行服务器。
1. 使用DNS spoofing将aircat.phicomm.com解析到服务器ip。