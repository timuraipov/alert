package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/logger"
	"go.uber.org/zap"
)

type MetricsCollector struct {
	mx                  sync.Mutex
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
	op := "agent.Send"
	m.mx.Lock()
	defer m.mx.Unlock()
	for key, val := range m.GaugeMetrics {
		typedValue, err := convertToFloat64(val)
		if err != nil {
			logger.Log.Debug("failed to Convert GaugeMetrics value",
				zap.String("operation", op),
				zap.String("value", fmt.Sprintf("%v", val)),
			)
		}

		metric := metric.Metrics{
			ID:    key,
			MType: metric.MetricTypeGauge,
			Value: &typedValue,
		}
		err = m.sendMetric(url, metric)
		if err != nil {
			logger.Log.Error("failed to  send metrics",
				zap.String("operation", op),
				zap.Error(err),
			)
			return err
		}
	}
	//send PollCount

	metric := metric.Metrics{
		ID:    "PollCount",
		MType: metric.MetricTypeCounter,
		Delta: &m.PollCount,
	}
	err := m.sendMetric(url, metric)
	if err != nil {
		return err
	}
	m.PollCount = 0
	return nil
}
func (m *MetricsCollector) sendMetric(url string, metricObj metric.Metrics) error {
	requestBody, err := json.Marshal(metricObj)
	if err != nil {
		log.Print(err)
	}
	res, err := http.Post(url, `application/json`, bytes.NewReader(requestBody))
	if err != nil {
		return err
	} else {
		res.Body.Close()
	}
	return nil
}
func (m *MetricsCollector) Run() {
	op := "agent.Run"
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
		err := m.Send("http://" + m.Addr + "/update/")
		if err != nil {
			logger.Log.Error("failed to Marshal body",
				zap.String("operation", op),
				zap.Error(err),
			)
			log.Print(err)
		}
		time.Sleep(time.Duration(m.ReportCountInterval) * time.Second)
	}

}
func convertToFloat64(value interface{}) (float64, error) {
	switch i := value.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	default:
		return math.NaN(), errors.New("getFloat: unknown value is of incompatible type")
	}
}
