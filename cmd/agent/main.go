package main

import (
	"github.com/timuraipov/alert/internal/agent"
)

var PollCount int64

func main() {
	agent := agent.New()
	agent.Run()
}

// func collect() map[string]interface{} {
// 	PollCount++
// 	gaugeMetricName := map[string]any{}
// 	var m runtime.MemStats
// 	runtime.ReadMemStats(&m)
// 	gaugeMetricName["Alloc"] = m.Alloc
// 	gaugeMetricName["BuckHashSys"] = m.BuckHashSys
// 	gaugeMetricName["Frees"] = m.Frees
// 	gaugeMetricName["GCCPUFraction"] = m.GCCPUFraction
// 	gaugeMetricName["GCSys"] = m.GCSys
// 	gaugeMetricName["HeapAlloc"] = m.HeapAlloc
// 	gaugeMetricName["HeapIdle"] = m.HeapIdle
// 	gaugeMetricName["HeapObjects"] = m.HeapObjects
// 	gaugeMetricName["HeapInuse"] = m.HeapInuse
// 	gaugeMetricName["HeapReleased"] = m.HeapReleased
// 	gaugeMetricName["HeapSys"] = m.HeapSys
// 	gaugeMetricName["LastGC"] = m.LastGC
// 	gaugeMetricName["Lookups"] = m.Lookups
// 	gaugeMetricName["MCacheInuse"] = m.MCacheInuse
// 	gaugeMetricName["MCacheSys"] = m.MCacheSys
// 	gaugeMetricName["MSpanInuse"] = m.MSpanInuse
// 	gaugeMetricName["MSpanSys"] = m.MSpanSys
// 	gaugeMetricName["Mallocs"] = m.Mallocs
// 	gaugeMetricName["NextGC"] = m.NextGC
// 	gaugeMetricName["NumForcedGC"] = m.NumForcedGC
// 	gaugeMetricName["NumGC"] = m.NumGC
// 	gaugeMetricName["OtherSys"] = m.OtherSys
// 	gaugeMetricName["PauseTotalNs"] = m.PauseTotalNs
// 	gaugeMetricName["StackInuse"] = m.StackInuse
// 	gaugeMetricName["StackSys"] = m.StackSys
// 	gaugeMetricName["Sys"] = m.Sys
// 	gaugeMetricName["TotalAlloc"] = m.TotalAlloc
// 	gaugeMetricName["RandomValue"] = rand.Float64()

// 	return gaugeMetricName
// 	// time.Sleep(2 * time.Second)
// }

//func GetMetricData(metricName string)
