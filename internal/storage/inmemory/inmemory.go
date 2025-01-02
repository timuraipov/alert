package inmemory

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/filestorage"
	"github.com/timuraipov/alert/internal/logger"
	"go.uber.org/zap"
)

type InMemory struct {
	mx              sync.RWMutex
	fileStorage     *filestorage.Storage
	DBGauge         map[string]float64
	DBCounter       map[string]int64
	config          *config.Config
	IsNeedSyncFlush bool
}

func (i *InMemory) init() {
	if i.config.Restore {
		i.load()
	}
	if i.config.StoreInterval == 0 {
		i.IsNeedSyncFlush = true
	} else {
		tickerFlushToDisk := time.NewTicker(time.Duration(i.config.StoreInterval) * time.Second)
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-tickerFlushToDisk.C:
					i.Flush()
				case <-done:
					tickerFlushToDisk.Stop()
					return
				}
			}
		}()
	}
}
func (i *InMemory) Flush() error {
	logger.Log.Debug(
		"called flush to disk method", zap.String("op", "op"),
	)
	metrics := i.GetAll()
	if len(metrics) > 0 {
		data, err := json.Marshal(metrics)
		if err != nil {
			return fmt.Errorf("failed to Marshal metrics %w", err)
		}
		err = i.fileStorage.Write(data)
		if err != nil {
			return fmt.Errorf("failed write metrics to disk %w", err)
		}
	}
	return nil
}
func (i *InMemory) load() error {
	metrics := make([]metric.Metrics, 0)
	data, err := i.fileStorage.Read()
	if err != nil {
		logger.Log.Error("failed to load file", zap.Error(err), zap.String("data", string(data)))
	}
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return fmt.Errorf("failed to load file %w", err)
	}
	for _, parsedMetric := range metrics {
		_, err = i.Save(parsedMetric)
		if err != nil {
			logger.Log.Error("failed to save metric", zap.Error(err))
		}
	}
	return nil
}

func (i *InMemory) Save(metricObj metric.Metrics) (metric.Metrics, error) {
	i.mx.Lock()
	defer i.mx.Unlock()
	responseMetric := metric.Metrics{}
	switch metricObj.MType {
	case metric.MetricTypeCounter:
		i.DBCounter[metricObj.ID] += *metricObj.Delta
		val := i.DBCounter[metricObj.ID]
		responseMetric.Delta = &val
	case metric.MetricTypeGauge:
		i.DBGauge[metricObj.ID] = *metricObj.Value
		val := i.DBGauge[metricObj.ID]
		responseMetric.Value = &val
	default:
		return metric.Metrics{}, fmt.Errorf("incorrect type or value")
	}
	responseMetric.ID = metricObj.ID
	responseMetric.MType = metricObj.MType
	if i.IsNeedSyncFlush {
		err := i.Flush()
		if err != nil {
			logger.Log.Error("can't flush metrics on disk", zap.Error(err))
		}
	}
	return responseMetric, nil
}
func (i *InMemory) GetAll() []metric.Metrics {
	i.mx.RLock()
	defer i.mx.RUnlock()
	var metrics []metric.Metrics
	for key, val := range i.DBGauge {
		metrics = append(metrics, metric.Metrics{
			ID:    key,
			MType: metric.MetricTypeGauge,
			Value: &val,
		})
	}
	for key, val := range i.DBCounter {
		metrics = append(metrics, metric.Metrics{
			ID:    key,
			MType: metric.MetricTypeCounter,
			Delta: &val,
		})
	}
	return metrics
}
func (i *InMemory) GetByTypeAndName(metricType, metricName string) (metric.Metrics, bool) {
	i.mx.RLock()
	defer i.mx.RUnlock()
	var foundMetrics metric.Metrics
	foundMetrics.ID = metricName
	foundMetrics.MType = metricType
	if metricType == metric.MetricTypeCounter {
		val, ok := i.DBCounter[metricName]
		if ok {
			foundMetrics.Delta = &val
			return foundMetrics, ok
		}
	}
	if metricType == metric.MetricTypeGauge {
		val, ok := i.DBGauge[metricName]
		if ok {
			foundMetrics.Value = &val
			return foundMetrics, ok
		}
	}
	return metric.Metrics{}, false
}
func New(fileStorage *filestorage.Storage, cfg *config.Config) (*InMemory, error) {
	dbGauge := make(map[string]float64)
	dbCounter := make(map[string]int64)
	return &InMemory{
		DBGauge:     dbGauge,
		DBCounter:   dbCounter,
		fileStorage: fileStorage,
		config:      cfg,
	}, nil
}
