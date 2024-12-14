package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type MetricsCollector struct {
	mx                  sync.RWMutex
	GaugeMetrics        map[string]interface{}
	PollInterval        int64
	ReportCountInterval int64
	Addr                string
	PollCount           int64
}

func New(flagRunAddr string, reportInterval, pollInterval int64) *MetricsCollector {
	return &MetricsCollector{
		GaugeMetrics:        map[string]interface{}{},
		PollInterval:        pollInterval,
		ReportCountInterval: reportInterval,
		Addr:                flagRunAddr,
		PollCount:           0,
	}
}
func (m *MetricsCollector) UpdateMetrics() {
	var memStat runtime.MemStats
	runtime.ReadMemStats(&memStat)
	m.mx.Lock()
	defer m.mx.Unlock()
	m.PollCount++
	m.GaugeMetrics["Alloc"] = memStat.Alloc
	m.GaugeMetrics["BuckHashSys"] = memStat.BuckHashSys
	m.GaugeMetrics["Frees"] = memStat.Frees
	m.GaugeMetrics["GCCPUFraction"] = memStat.GCCPUFraction
	m.GaugeMetrics["GCSys"] = memStat.GCSys
	m.GaugeMetrics["HeapAlloc"] = memStat.HeapAlloc
	m.GaugeMetrics["HeapIdle"] = memStat.HeapIdle
	m.GaugeMetrics["HeapObjects"] = memStat.HeapObjects
	m.GaugeMetrics["HeapInuse"] = memStat.HeapInuse
	m.GaugeMetrics["HeapReleased"] = memStat.HeapReleased
	m.GaugeMetrics["HeapSys"] = memStat.HeapSys
	m.GaugeMetrics["LastGC"] = memStat.LastGC
	m.GaugeMetrics["Lookups"] = memStat.Lookups
	m.GaugeMetrics["MCacheInuse"] = memStat.MCacheInuse
	m.GaugeMetrics["MCacheSys"] = memStat.MCacheSys
	m.GaugeMetrics["MSpanInuse"] = memStat.MSpanInuse
	m.GaugeMetrics["MSpanSys"] = memStat.MSpanSys
	m.GaugeMetrics["Mallocs"] = memStat.Mallocs
	m.GaugeMetrics["NextGC"] = memStat.NextGC
	m.GaugeMetrics["NumForcedGC"] = memStat.NumForcedGC
	m.GaugeMetrics["NumGC"] = memStat.NumGC
	m.GaugeMetrics["OtherSys"] = memStat.OtherSys
	m.GaugeMetrics["PauseTotalNs"] = memStat.PauseTotalNs
	m.GaugeMetrics["StackInuse"] = memStat.StackInuse
	m.GaugeMetrics["StackSys"] = memStat.StackSys
	m.GaugeMetrics["Sys"] = memStat.Sys
	m.GaugeMetrics["TotalAlloc"] = memStat.TotalAlloc
	m.GaugeMetrics["RandomValue"] = rand.Float64()
}
func (m *MetricsCollector) Send(url string) error {
	m.mx.RLock()
	defer m.mx.RUnlock()
	for key, val := range m.GaugeMetrics {
		err := m.sendMetric(url, "gauge", key, val)
		if err != nil {
			return err
		}
	}
	//send PollCount
	err := m.sendMetric(url, "counter", "PollCount", m.PollCount)
	if err != nil {
		return err
	}
	m.PollCount = 0
	return nil
}
func (m *MetricsCollector) sendMetric(url, metricType, metricName string, metricValue interface{}) error {
	fullPath := url + "/update/" + metricType + "/" + metricName + "/" + fmt.Sprintf("%v", metricValue)
	req, err := http.NewRequest(http.MethodPost, fullPath, nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	} else {
		res.Body.Close()
	}
	return nil
}
func (m *MetricsCollector) Run() {
	tickerUpdateMetrics := time.NewTicker(time.Duration(m.PollInterval) * time.Second)
	quitUpdateMetrics := make(chan struct{})
	go func() {
		for {
			select {
			case <-tickerUpdateMetrics.C:
				m.UpdateMetrics()
			case <-quitUpdateMetrics:
				tickerUpdateMetrics.Stop()
				return
			}
		}
	}()
	time.Sleep(time.Duration(m.ReportCountInterval) * time.Second)
	for {
		err := m.Send("http://" + m.Addr)
		if err != nil {
			log.Print(err)
		}
		time.Sleep(time.Duration(m.ReportCountInterval) * time.Second)
	}

}
