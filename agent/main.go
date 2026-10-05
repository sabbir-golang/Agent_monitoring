package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	netio "github.com/shirou/gopsutil/v4/net"
)

func GetIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return "unknown"
}
func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
func Ram() float64 {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return vm.UsedPercent
}
func CPU() float64 {
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Println(err)
	}
	return cpuPercent[0]
}
func Disk() float64 {
	diskusg, err := disk.Usage("/")
	if err != nil {
		return 0
	}
	return diskusg.UsedPercent
}

func Net() float64 {
	netusg, err := netio.IOCounters(false)
	if err != nil {
		panic(err)
	}
	return float64(netusg[0].BytesSent) / float64(netusg[0].BytesRecv)
}

type Usage struct {
	IP       string  `json:"ip"`
	Hostname string  `json:"hostname"`
	CPU      float64 `json:"cpu"`
	RAM      float64 `json:"ram"`
	Disk     float64 `json:"disk"`
	Upload   float64 `json:"upload"`
	Download float64 `json:"download"`
}

func main() {
	conn, _, _ := websocket.DefaultDialer.Dial(
		"ws://localhost:8080/ws",
		nil,
	)
	defer conn.Close()
	ip := GetIP()
	host := Hostname()
	for {
		download := Net()
		upload := Net()
		usg := Usage{
			IP:       ip,
			Hostname: host,
			CPU:      CPU(),
			RAM:      Ram(),
			Disk:     Disk(),
			Upload:   upload,
			Download: download,
		}
		conn.WriteJSON(usg)
		time.Sleep(2 * time.Second)
	}

}
