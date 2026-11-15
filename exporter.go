package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"syscall"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

var (
	cpuUsageHist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "system_cpu_usage_percent",
		Help:    "Histogram of CPU usage samples (percent)",
		Buckets: prometheus.LinearBuckets(0, 10, 11),
	})

	memUsageHist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "system_memory_usage_percent",
		Help:    "Current memory usage percentage",
		Buckets: prometheus.LinearBuckets(0, 10, 11),
	})

	netRecv = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "system_network_receive_mbytes_total",
		Help: "Total received network mbytes across all interfaces",
	})

	netSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "system_network_transmit_mbytes_total",
		Help: "Total transmitted network mbytes across all interfaces",
	})

	powerAC = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_power_on_ac",
		Help: "1 if plugged in, 0 if on battery",
	})

	netRecvPrev = uint64(0)
	netSentPrev = uint64(0)
)

func updateDelta(prev *uint64, current uint64) (delta uint64) {
	if *prev > 0 {
		delta = current - *prev
	} else {
		delta = 0
	}
	*prev = current
	return delta
}

func collectMetrics() {
	// CPU usage
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		cpuUsageHist.Observe(percents[0])
	}

	// Memory usage
	if vm, err := mem.VirtualMemory(); err == nil {
		memUsageHist.Observe(vm.UsedPercent)
	}

	// Network stats
	if ioStats, err := net.IOCounters(false); err == nil && len(ioStats) > 0 {

		netRecv.Add(float64(updateDelta(&netRecvPrev, ioStats[0].BytesRecv)) / (1024 * 1024))
		netSent.Add(float64(updateDelta(&netSentPrev, ioStats[0].BytesSent)) / (1024 * 1024))
	}

	// Power (зарядка)
	//if sensors, err := host.SensorsTemperatures(); err == nil {
	//	_ = sensors // просто проверка, библиотека работает
	//}
	// gopsutil не имеет прямого метода для питания, используем Windows API
	onAC := checkACPower()
	if onAC {
		powerAC.Set(1)
	} else {
		powerAC.Set(0)
	}
}

type SYSTEM_POWER_STATUS struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// checkACPower — определяет, подключено ли питание (Windows)
func checkACPower() bool {
	// Используем WinAPI через syscall
	// Для упрощения — читаем из системной функции GetSystemPowerStatus
	var sps SYSTEM_POWER_STATUS
	mod := syscall.NewLazyDLL("kernel32.dll")
	proc := mod.NewProc("GetSystemPowerStatus")
	proc.Call(uintptr(unsafe.Pointer(&sps)))
	return sps.ACLineStatus == 1
}

func runExporterNormal() {
	ip := flag.String("ip", "0.0.0.0", "IP address to bind the exporter")
	port := flag.Int("port", 12341, "Port to expose metrics on")
	flag.Parse()
	listenAddr := fmt.Sprintf("%s:%d", *ip, *port)

	r := prometheus.NewRegistry()

	r.MustRegister(cpuUsageHist, memUsageHist, netRecv, netSent, powerAC) // printerPages, printerQueueJobs

	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	})

	log.Printf("Exporter running on %s/metrics\r\n", listenAddr)
	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic(err)
	}
}
