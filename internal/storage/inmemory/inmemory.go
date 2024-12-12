package inmemory

import (
	"fmt"
	"sync"

	"github.com/timuraipov/alert/internal/domain/metric"
)

type InMemory struct {
	mx        sync.Mutex
	DBGauge   map[string]float64
	DBCounter map[string]int64
}

func (i *InMemory) Save(metricObj metric.Metric) error {
	i.mx.Lock()
	defer i.mx.Unlock()
	switch metricObj.Type {
	case metric.MetricTypeCounter:
		i.DBCounter[metricObj.Name] += metricObj.ValueCounter
	case metric.MetricTypeGauge:
		i.DBGauge[metricObj.Name] = metricObj.ValueGauge
	default:
		return fmt.Errorf("incorrect type or value")
	}
	return nil
}
func (i *InMemory) GetAll() map[string]interface{} {
	metrics := make(map[string]interface{})
	for key, val := range i.DBGauge {
		metrics[key] = val
	}
	for key, val := range i.DBCounter {
		metrics[key] = val
	}
	return metrics
}
func (i *InMemory) GetByTypeAndName(metricType, metricName string) (interface{}, bool) {
	if metricType == metric.MetricTypeCounter {
		val, ok := i.DBCounter[metricName]
		return val, ok
	}
	if metricType == metric.MetricTypeGauge {
		val, ok := i.DBGauge[metricName]
		return val, ok
	}
	return nil, false
}
func New() (*InMemory, error) {
	dbGauge := make(map[string]float64)
	dbCounter := make(map[string]int64)
	return &InMemory{
		DBGauge:   dbGauge,
		DBCounter: dbCounter,
	}, nil
}
