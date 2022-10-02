package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"battery_exporter/battery"
)

var g_BatteryHandles = map[string]*battery.Handle{}

var (
	powerStateGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_power_state",
			Help: "battery state in bitmask (1: power on line, 2: discharging, 4: charging, 8: critical)",
		},
		[]string{"id"})
	temperatureGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_temperature",
			Help: "battery temperature in Celsius",
		},
		[]string{"id"})
	designedCapacityGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_designed_capacity",
			Help: "battery designed capacity in Wh",
		},
		[]string{"id"})
	availableGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_available",
			Help: "battery available capacity in Wh",
		},
		[]string{"id"})
	fullChargedCapacityGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_full_charged_capacity",
			Help: "battery full charged capacity in Wh",
		},
		[]string{"id"})
	voltageGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_voltage",
			Help: "battery voltage",
		},
		[]string{"id"})
	cycleCountGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_cycle",
			Help: "battery cycle count",
		},
		[]string{"id"})
	rateGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "battery_rate",
			Help: "battery rate in Watt (+: charging, -: discharging)",
		},
		[]string{"id"})
)

func discoverBatteries() {
	// 10 is no basis
	for i := 0; i < 10; i++ {
		handle, err := battery.OpenBatteryHandle(i)
		if err != nil {
			if len(g_BatteryHandles) == 0 {
				log.Printf("[warning] OpenBatteryHandle(%d): %s", i, err)
			}
			continue
		}

		id, err := battery.GetBatteryUniqueId(handle)
		if err != nil {
			log.Printf("[warning] GetBatteryUniqueId: %s. Use %d", err, i)
			id = strconv.Itoa(i)
		}

		log.Printf("[info] found battery \"%s\"", id)
		g_BatteryHandles[id] = handle
	}
}

func collectBattery() {
	for id, handle := range g_BatteryHandles {
		info, err := battery.GetBatteryInfo(handle)
		if err != nil {
			log.Printf("[error] GetBatteryInfo: %s", err)
		} else {
			designedCapacityGauge.With(prometheus.Labels{"id": id}).Set(float64(info.DesignedCapacity) / 1000.0)
			fullChargedCapacityGauge.With(prometheus.Labels{"id": id}).Set(float64(info.FullChargedCapacity) / 1000.0)
			cycleCountGauge.With(prometheus.Labels{"id": id}).Set(float64(info.CycleCount))
		}

		status, err := battery.GetBatteryStatus(handle)
		if err != nil {
			log.Printf("[error] GetBatteryStatus: %s", err)
		} else {
			powerStateGauge.With(prometheus.Labels{"id": id}).Set(float64(status.PowerState))
			availableGauge.With(prometheus.Labels{"id": id}).Set(float64(status.Capacity) / 1000.0)
			voltageGauge.With(prometheus.Labels{"id": id}).Set(float64(status.Voltage) / 1000.0)
			rateGauge.With(prometheus.Labels{"id": id}).Set(float64(status.Rate) / 1000.0)
		}

		temp, err := battery.GetBatteryTemperature(handle)
		if err != nil {
			log.Printf("[error] GetBatteryTemperature: %s", err)
		} else {
			temperatureGauge.With(prometheus.Labels{"id": id}).Set(temp)
		}
	}
}

func collect() {
	collectBattery()
}

func main() {
	listen := flag.String("listen", ":9004", "metrics listen address")
	flag.Parse()

	// discover batteries
	discoverBatteries()

	// define metrics
	prometheus.MustRegister(temperatureGauge)
	prometheus.MustRegister(designedCapacityGauge)
	prometheus.MustRegister(fullChargedCapacityGauge)
	prometheus.MustRegister(cycleCountGauge)
	prometheus.MustRegister(powerStateGauge)
	prometheus.MustRegister(availableGauge)
	prometheus.MustRegister(voltageGauge)
	prometheus.MustRegister(rateGauge)
	prometheus.MustRegister(prometheus.NewBuildInfoCollector())

	// start HTTP server
	log.Printf("[info] listening %s", *listen)
	promhandler := promhttp.Handler()
	http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		collect()
		promhandler.ServeHTTP(w, r)
	}))
	log.Fatal(http.ListenAndServe(*listen, nil))
}
