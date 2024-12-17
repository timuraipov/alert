package inmemory

import (
	"fmt"
	"sync"

	"github.com/timuraipov/alert/internal/domain/metric"
)

type InMemory struct {
	mx        sync.RWMutex
	DBGauge   map[string]float64
	DBCounter map[string]int64
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
func New() (*InMemory, error) {
	dbGauge := make(map[string]float64)
	dbCounter := make(map[string]int64)
	return &InMemory{
		DBGauge:   dbGauge,
		DBCounter: dbCounter,
	}, nil
}
