package inmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/filestorage"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/storage"
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

func (i *InMemory) init() error {
	if i.config.Restore {
		err := i.load()
		if err != nil {
			return err
		}
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
	return nil
}
func (i *InMemory) Flush() error {
	logger.Log.Debug(
		"called flush to disk method", zap.String("op", "op"),
	)
	metrics, err := i.GetAll(context.Background())
	if err != nil {
		return err
	}
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
		_, err = i.Save(context.Background(), parsedMetric)
		if err != nil {
			logger.Log.Error("failed to save metric", zap.Error(err))
		}
	}
	return nil
}

func (i *InMemory) Save(ctx context.Context, metricObj metric.Metrics) (metric.Metrics, error) {
	responseList, err := i.save([]metric.Metrics{metricObj})
	if err != nil {
		return metric.Metrics{}, err
	}
	return responseList[0], nil
}
func (i *InMemory) GetAll(ctx context.Context) ([]metric.Metrics, error) {
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
	return metrics, nil
}
func (i *InMemory) GetByTypeAndName(ctx context.Context, metricType, metricName string) (metric.Metrics, error) {
	i.mx.RLock()
	defer i.mx.RUnlock()
	var foundMetrics metric.Metrics
	foundMetrics.ID = metricName
	foundMetrics.MType = metricType
	if metricType == metric.MetricTypeCounter {
		val, ok := i.DBCounter[metricName]
		if ok {
			foundMetrics.Delta = &val
			return foundMetrics, nil
		}
	}
	if metricType == metric.MetricTypeGauge {
		val, ok := i.DBGauge[metricName]
		if ok {
			foundMetrics.Value = &val
			return foundMetrics, nil
		}
	}
	return metric.Metrics{}, storage.ErrMetricNotFound
}
func New(fileStorage *filestorage.Storage, cfg *config.Config) (*InMemory, error) {
	dbGauge := make(map[string]float64)
	dbCounter := make(map[string]int64)
	storage := &InMemory{
		DBGauge:     dbGauge,
		DBCounter:   dbCounter,
		fileStorage: fileStorage,
		config:      cfg,
	}
	err := storage.init()
	if err != nil {
		return nil, err
	}
	return storage, nil
}
func (i *InMemory) SaveBatch(ctx context.Context, metrics []metric.Metrics) error {
	_, err := i.save(metrics)
	if err != nil {
		return err
	}
	return nil
}

func (i *InMemory) save(metricsList []metric.Metrics) ([]metric.Metrics, error) {
	var resultList []metric.Metrics
	i.mx.RLock()
	defer i.mx.RUnlock()
	for _, m := range metricsList {
		responseMetric := metric.Metrics{}

		switch m.MType {
		case metric.MetricTypeCounter:
			i.DBCounter[m.ID] += *m.Delta
			val := i.DBCounter[m.ID]
			responseMetric.Delta = &val
		case metric.MetricTypeGauge:
			i.DBGauge[m.ID] = *m.Value
			val := i.DBGauge[m.ID]
			responseMetric.Value = &val
		default:
			return nil, fmt.Errorf("incorrect type or value")
		}
		responseMetric.ID = m.ID
		responseMetric.MType = m.MType
		resultList = append(resultList, responseMetric)
	}

	if i.IsNeedSyncFlush { // maybe need to move into loop
		err := i.Flush()
		if err != nil {
			logger.Log.Error("can't flush metrics on disk", zap.Error(err))
		}
	}
	return resultList, nil
}
func (i *InMemory) Ping(ctx context.Context) error {
	return nil
}
