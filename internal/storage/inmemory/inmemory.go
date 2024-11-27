package inmemory

import (
	"fmt"

	"github.com/timuraipov/alert/internal/http-server/handlers/metrics"
)

type InMemory struct {
	DBGauge   map[string]float64
	DBCounter map[string]int64
}

func (i *InMemory) Save(metric metrics.Metric) error {
	switch metric.Type {
	case metrics.MetricTypeCounter:
		i.DBCounter[metric.Name] += metric.ValueCounter
	case metrics.MetricTypeGauge:
		i.DBGauge[metric.Name] = metric.ValueGauge
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
