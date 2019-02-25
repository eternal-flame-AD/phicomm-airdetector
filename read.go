package airdetector

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net"
	"strconv"
	"time"
)

type packet []byte

type pType byte

const (
	connect pType = 0x03
)

var tail = [6]byte{0xff, 0x23, 0x45, 0x4e, 0x44, 0x23}

type rawReading struct {
	Humidity    string `json:"humidity"`
	Temperature string `json:"temperature"`
	HCHO        string `json:"hcho"`
	PM25        string `json:"value"`
}

func (c rawReading) Reading() (res Reading, err error) {
	res.Humidity, err = strconv.ParseFloat(c.Humidity, 64)
	res.Temperature, err = strconv.ParseFloat(c.Temperature, 64)
	res.HCHO, err = strconv.ParseFloat(c.HCHO, 64)
	res.HCHO /= 1000.
	res.PM25, err = strconv.Atoi(c.PM25)
	return
}

// Reading is a device reading.
type Reading struct {
	Humidity    float64
	Temperature float64
	HCHO        float64
	PM25        int
}

func (c packet) MacAddr() (res [6]byte) {
	copy(res[:], c[0x11:0x17])
	return
}

func (c packet) Type() pType {
	return pType(c[0x18])
}

func (c packet) IsReading() bool {
	t := c.Type()
	return t >= 0x4e && t <= 0x50
}

func (c packet) IsValid() bool {
	return bytes.Equal(c[len(c)-len(tail):], tail[:])
}

func (c packet) Reading() (*Reading, error) {
	if !c.IsReading() {
		return nil, errors.New("attempt to get reading from a non-measurement")
	}
	res := new(rawReading)
	err := json.Unmarshal(c[0x1c:len(c)-len(tail)], res)
	if err != nil {
		return nil, err
	}
	read, err := res.Reading()
	if err != nil {
		return nil, err
	}
	return &read, nil
}

type deviceConnection struct {
	conn      *net.TCPConn
	deviceMAC [6]byte
}

// ReadingWithConnInfo is a reading with connection and device info included.
type ReadingWithConnInfo struct {
	Reading
	DeviceMAC  [6]byte
	RemoteAddr net.Addr
}

func (c deviceConnection) handle(output chan<- ReadingWithConnInfo) {
	buf := make([]byte, 1024)
	for {
		c.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		l, err := c.conn.Read(buf)
		if err != nil {
			log.Println(err)
			c.conn.Close()
			return
		}

		data := packet(buf[:l])
		if !data.IsValid() {
			log.Printf("received invalid packet: %x", data)
			return
		}

		if data.Type() == connect {
			c.deviceMAC = data.MacAddr()
		} else if c.deviceMAC != data.MacAddr() {
			log.Printf("Received data packet of inconsistent mac address: expected %x got %x", c.deviceMAC, data.MacAddr())
		}
		if t := data.Type(); t != connect && !data.IsReading() {
			log.Printf("Received unknown data packet of type %x, len %d", data.Type(), len(data))
		}

		if data.IsReading() {
			reading, err := data.Reading()
			if err != nil {
				log.Println(err)
			}
			output <- ReadingWithConnInfo{
				Reading:    *reading,
				DeviceMAC:  c.deviceMAC,
				RemoteAddr: c.conn.RemoteAddr(),
			}
		}

	}
}

// Listen starts listening for connections
func Listen() (<-chan ReadingWithConnInfo, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":9000")
	if err != nil {
		return nil, err
	}
	n, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	output := make(chan ReadingWithConnInfo)
	go func() {
		for {
			c, err := n.AcceptTCP()
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("Got connection from %s", c.RemoteAddr().String())
			conn := deviceConnection{
				conn: c,
			}
			go conn.handle(output)
		}
	}()
	return output, nil
}
