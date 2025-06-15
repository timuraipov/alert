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
	"strconv"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/timuraipov/alert/internal/agent/config"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/pkg/hmac"
	"go.uber.org/zap"
)

const (
	SignatureHeaderName = "HashSHA256"
	NumJobs             = 5
)

type MetricsCollector struct {
	mx           sync.Mutex
	GaugeMetrics map[string]interface{}
	PollCount    int64
	cfg          *config.Config
	jobs         chan metric.Metrics
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

func (m *MetricsCollector) AdditionalMetrics() {
	v, _ := mem.VirtualMemory()
	cpu, _ := cpu.Percent(time.Second, true)
	m.mx.Lock()
	defer m.mx.Unlock()
	m.GaugeMetrics["TotalMemory"] = v.Total
	m.GaugeMetrics["FreeMemory"] = v.Free
	for i, unit := range cpu {
		key := "CPUutilization" + strconv.Itoa(i)
		m.GaugeMetrics[key] = unit
	}
}

func (m *MetricsCollector) GetData() []metric.Metrics {
	op := "agent.GetData"
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
	}
	value := new(int64)
	*value = m.PollCount
	metric := metric.Metrics{
		ID:    "PollCount",
		MType: metric.MetricTypeCounter,
		Delta: value,
	}

	metrics = append(metrics, metric)
	return metrics
}

func (m *MetricsCollector) Send(url string) error {
	op := "agent.Send"
	_ = op
	metrics := m.GetData()
	if m.cfg.RateLimit > 0 {
		go func() {
			for _, metric := range metrics {
				m.jobs <- metric
			}
		}()
	} else {
		_, err := m.sendMetric(url, metrics)
		if err != nil {
			return err
		}
	}
	m.mx.Lock()
	m.PollCount = 0
	m.mx.Unlock()
	return nil
}

func (m *MetricsCollector) sendMetric(url string, metricObj []metric.Metrics) (int, error) {
	op := "agent.SendMetric"
	requestBody, err := json.Marshal(metricObj)
	if err != nil {
		log.Print(err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		log.Print(err)
	}
	if len(m.cfg.SignBodyKey) > 0 {
		header := hmac.SignData(requestBody, m.cfg.SignBodyKey)
		req.Header.Add(SignatureHeaderName, header)
	}
	for i := 0; i < retryCount; i++ {
		res, err := client.Do(req)

		if err == nil {
			res.Body.Close()
			return res.StatusCode, nil
		}

		logger.Log.Error("can't sent metrics",
			zap.String("operation", op),
			zap.String("trying to resend metrics:", fmt.Sprintf("tries number- %d", i+1)),
			zap.Error(err),
		)
		time.Sleep(time.Duration(retryInterval[i]) * time.Second)
	}
	return http.StatusInternalServerError, nil
}

func (m *MetricsCollector) Run() {
	op := "agent.Run"
	url := "http://" + m.cfg.ServerAddr + "/updates/"
	tickerUpdateMetrics := time.NewTicker(time.Duration(m.cfg.PollInterval) * time.Second)
	quitUpdateMetrics := make(chan struct{})
	if m.cfg.RateLimit > 0 {
		m.jobs = make(chan metric.Metrics, NumJobs)
		for w := 1; w <= m.cfg.RateLimit; w++ {
			go m.worker(w, url)
		}
	}
	go func() {
		for {
			select {
			case <-tickerUpdateMetrics.C:
				m.UpdateMetrics()
				go m.AdditionalMetrics()
			case <-quitUpdateMetrics:
				tickerUpdateMetrics.Stop()
				return
			}
		}
	}()
	time.Sleep(time.Duration(m.cfg.ReportInterval) * time.Second)
	for {
		err := m.Send(url)
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

func (m *MetricsCollector) worker(id int, url string) {
	op := "agent.Worker"
	// todo some work
	for job := range m.jobs {
		logger.Log.Info(fmt.Sprintf("worker with id %d", id))
		_, err := m.sendMetric(url, []metric.Metrics{job})
		if err != nil {
			logger.Log.Error("failed to Marshal body",
				zap.String("operation", op),
				zap.Error(err),
			)
		}
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
