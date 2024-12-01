package inmemory

import (
	"fmt"

	"github.com/timuraipov/alert/internal/domain/metric"
)

type InMemory struct {
	DBGauge   map[string]float64
	DBCounter map[string]int64
}

func (i *InMemory) Save(metricObj metric.Metric) error {
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

func New() (*InMemory, error) {
	dbGauge := make(map[string]float64)
	dbCounter := make(map[string]int64)
	return &InMemory{
		DBGauge:   dbGauge,
		DBCounter: dbCounter,
	}, nil
}
