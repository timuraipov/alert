package storage

import (
	"context"
	"errors"

	"github.com/timuraipov/alert/internal/domain/metric"
)

var (
	ErrMetricNotFound = errors.New("metric not found")
)

type DBStorage interface {
	Save(ctx context.Context, metric metric.Metrics) (metric.Metrics, error)
	SaveBatch(ctx context.Context, metrics []metric.Metrics) error
	GetAll(ctx context.Context) ([]metric.Metrics, error)
	GetByTypeAndName(ctx context.Context, metricType, metricName string) (metric.Metrics, error)
	Flush() error
}
type DBHealthStorage interface {
	Ping(ctx context.Context) error
}
