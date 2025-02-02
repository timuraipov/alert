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

	"github.com/timuraipov/alert/internal/agent/config"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/pkg/hmac"
	"go.uber.org/zap"
)

const SignatureHeaderName = "HashSHA256"

type MetricsCollector struct {
	mx           sync.Mutex
	GaugeMetrics map[string]interface{}
	PollCount    int64
	cfg          *config.Config
}

const retryCount = 3

var retryInterval = []int{1, 3, 5}

func New(cfg *config.Config) *MetricsCollector {
	return &MetricsCollector{
		GaugeMetrics: map[string]interface{}{},
		PollCount:    0,
		cfg:          cfg,
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
	var metrics []metric.Metrics
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
		metrics = append(metrics, metric)
		// err = m.sendMetric(url, metric)
		// if err != nil {
		// 	logger.Log.Error("failed to  send metrics",
		// 		zap.String("operation", op),
		// 		zap.Error(err),
		// 	)
		// 	return err
		// }
	}
	//send PollCount

	metric := metric.Metrics{
		ID:    "PollCount",
		MType: metric.MetricTypeCounter,
		Delta: &m.PollCount,
	}

	metrics = append(metrics, metric)
	statusCode, err := m.sendMetric(url, metrics)
	if statusCode == http.StatusRequestTimeout || statusCode >= http.StatusInternalServerError {
		logger.Log.Error("can't save metrics",
			zap.String("operation", op),
			zap.String("trying to resend metrics:", "start to retry"),
			zap.Error(err),
		)
		for i := 0; i < retryCount; i++ {
			time.Sleep(time.Duration(retryInterval[i]) * time.Second)
			statusCode, err = m.sendMetric(url, metrics)
			if !(err != nil || statusCode == http.StatusRequestTimeout || statusCode >= http.StatusInternalServerError) {
				break
			}
			logger.Log.Error("can't save metrics",
				zap.String("operation", op),
				zap.String("trying to resend metrics:", fmt.Sprintf("tries number- %d", i+1)),
				zap.Error(err),
			)
		}
	}
	if err != nil {
		return err
	}
	m.PollCount = 0
	return nil
}
func (m *MetricsCollector) sendMetric(url string, metricObj []metric.Metrics) (int, error) {
	requestBody, err := json.Marshal(metricObj)
	if err != nil {
		log.Print(err)
	}

	//res, err := http.Post(url, `application/json`, bytes.NewReader(requestBody))
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		log.Print(err)
	}
	if len(m.cfg.SignBodyKey) > 0 {
		header := hmac.SignData(requestBody, m.cfg.SignBodyKey)
		req.Header.Add(SignatureHeaderName, header)
	}
	res, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, err
	} else {
		res.Body.Close()
	}
	return res.StatusCode, nil
}
func (m *MetricsCollector) Run() {
	op := "agent.Run"
	tickerUpdateMetrics := time.NewTicker(time.Duration(m.cfg.PollInterval) * time.Second)
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
	time.Sleep(time.Duration(m.cfg.ReportInterval) * time.Second)
	for {
		err := m.Send("http://" + m.cfg.ServerAddr + "/updates/")
		if err != nil {
			logger.Log.Error("failed to Marshal body",
				zap.String("operation", op),
				zap.Error(err),
			)
			log.Print(err)
		}
		time.Sleep(time.Duration(m.cfg.ReportInterval) * time.Second)
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
