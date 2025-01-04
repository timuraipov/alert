package inmemory

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timuraipov/alert/internal/common"
	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/filestorage"
	commonStorage "github.com/timuraipov/alert/internal/storage"
)

func TestSaveGauge(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		metrics    []metric.Metrics
		want       float64
		metricType string
	}{
		{
			name: "positive Gauge",
			err:  nil,
			metrics: []metric.Metrics{
				{
					MType: metric.MetricTypeGauge,
					ID:    "someName",
					Value: common.Pointer(1.009),
				},
			},
			want: 1.009,
		},
		{
			name: "positive multi Gauge",
			err:  nil,
			metrics: []metric.Metrics{
				{
					MType: metric.MetricTypeGauge,
					ID:    "someName",
					Value: common.Pointer(1.001),
				},
				{
					MType: metric.MetricTypeGauge,
					ID:    "someName",
					Value: common.Pointer(12.009),
				},
			},
			want: 12.009,
		},
	}

	for _, test := range tests {

		saver, err := getStorage(nil)
		assert.NoError(t, err)
		var currentData metric.Metrics
		t.Run(test.name, func(t *testing.T) {
			for _, metric := range test.metrics {
				currentData, err = saver.Save(context.Background(), metric)
				assert.NoError(t, err)
			}

			assert.Equal(t, test.want, *(currentData.Value))
		})
	}
}

func TestSaveCounter(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		metrics    []metric.Metrics
		want       int64
		metricType string
	}{
		{
			name: "positive Counter 1 value",
			err:  nil,
			metrics: []metric.Metrics{
				{
					MType: metric.MetricTypeCounter,
					ID:    "someName",
					Delta: common.Pointer(int64(50)),
				},
			},
			want: 50,
		},
		{
			name: "positive Counter multi value",
			err:  nil,
			metrics: []metric.Metrics{
				{
					MType: metric.MetricTypeCounter,
					ID:    "someName",
					Delta: common.Pointer(int64(1)),
				},
				{
					MType: metric.MetricTypeCounter,
					ID:    "someName",
					Delta: common.Pointer(int64(2)),
				},
				{
					MType: metric.MetricTypeCounter,
					ID:    "someName",
					Delta: common.Pointer(int64(1000)),
				},
			},
			want: 1003,
		},
	}

	for _, test := range tests {
		saver, err := getStorage(nil)
		assert.NoError(t, err)
		var currentData metric.Metrics
		t.Run(test.name, func(t *testing.T) {
			for _, metric := range test.metrics {
				currentData, err = saver.Save(context.Background(), metric)
				assert.NoError(t, err)
			}

			assert.Equal(t, test.want, *(currentData.Delta))
		})
	}
}

func TestFileStorage(t *testing.T) {
	seed := []metric.Metrics{
		{
			MType: metric.MetricTypeCounter,
			ID:    "counterKey",
			Delta: common.Pointer(int64(50)),
		},
		{
			MType: metric.MetricTypeCounter,
			ID:    "counterKey",
			Delta: common.Pointer(int64(1)),
		},

		{
			MType: metric.MetricTypeGauge,
			ID:    "gaugeKey",
			Value: common.Pointer(2.01),
		},
		{
			MType: metric.MetricTypeGauge,
			ID:    "gaugeKey2",
			Value: common.Pointer(1000.001),
		},
	}
	tests := []struct {
		name  string
		want  string
		found bool
	}{
		{
			name: "positive load test",
			want: `[  {"id": "counterKey",
        "type": "counter",
        "delta": 51}, {"id": "gaugeKey",
        "type": "gauge",
        "value": 2.01}, {"id": "gaugeKey2",
        "type": "gauge",
        "value": 1000.001}]`,
			found: true,
		},
	}
	file, err := os.CreateTemp("", "temp")
	assert.NoError(t, err)
	defer os.Remove(file.Name())
	assert.NoError(t, err)
	cfg := &config.Config{
		StoreInterval:   0,
		FileStoragePath: file.Name(),
		Restore:         false,
	}
	storage, err := getStorage(cfg)
	assert.NoError(t, err)
	_, err = storage.save(seed)
	assert.NoError(t, err)
	newCfg := &config.Config{
		StoreInterval:   0,
		FileStoragePath: file.Name(),
		Restore:         true,
	}
	newStorage, err := getStorage(newCfg)
	assert.NoError(t, err)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metrics, err := newStorage.GetAll(context.Background())
			slices.SortFunc(metrics, func(a, b metric.Metrics) int {
				if n := strings.Compare(a.ID, b.ID); n != 0 {
					return n
				}
				return cmp.Compare[string](a.MType, b.MType)
			})
			assert.NoError(t, err)
			jsonMetrics, err := json.Marshal(metrics)
			assert.NoError(t, err)
			assert.JSONEqf(t, fmt.Sprint(test.want), fmt.Sprint(string(jsonMetrics)), "error message %s", "formatted")
		})
	}

}
func Seed(storage *InMemory) {
	seeds := []metric.Metrics{
		{
			MType: metric.MetricTypeCounter,
			ID:    "counterKey",
			Delta: common.Pointer(int64(50)),
		},
		{
			MType: metric.MetricTypeCounter,
			ID:    "counterKey",
			Delta: common.Pointer(int64(1)),
		},

		{
			MType: metric.MetricTypeGauge,
			ID:    "gaugeKey",
			Value: common.Pointer(2.01),
		},
		{
			MType: metric.MetricTypeGauge,
			ID:    "gaugeKey2",
			Value: common.Pointer(1000.001),
		},
	}

	for _, seed := range seeds {
		_, err := storage.Save(context.Background(), seed)
		if err != nil {
			panic(err)
		}
	}
}
func TestGetAll(t *testing.T) {
	tests := []struct {
		name  string
		want  metric.Metrics
		found bool
	}{
		{
			name: "positive test counter",
			want: metric.Metrics{
				ID:    "counterKey",
				MType: metric.MetricTypeCounter,
				Delta: common.Pointer(int64(51)),
			},
			found: true,
		},
		{
			name: "positive test gauge",
			want: metric.Metrics{
				ID:    "gaugeKey",
				MType: metric.MetricTypeGauge,
				Value: common.Pointer(2.01),
			},
			found: true,
		},
		{
			name: "positive test gauge case2",
			want: metric.Metrics{
				ID:    "gaugeKey2",
				MType: metric.MetricTypeGauge,
				Value: common.Pointer(1000.001),
			},
			found: true,
		},
		{
			name: "negative test counter ",
			want: metric.Metrics{
				ID:    "counterKeyNotExist",
				MType: metric.MetricTypeCounter,
				Value: common.Pointer(1000.001),
			},
			found: false,
		},
		{
			name: "negative test gauge",
			want: metric.Metrics{
				ID:    "gaugeKeyNotExist",
				MType: metric.MetricTypeCounter,
				Value: common.Pointer(1000.001),
			},
			found: false,
		},
	}
	storage, err := getStorage(nil)
	assert.NoError(t, err)
	Seed(storage)
	contains := func(slice []metric.Metrics, value metric.Metrics) bool {
		for _, elem := range slice {
			if elem.ID == value.ID && elem.MType == value.MType {
				switch value.MType {
				case metric.MetricTypeCounter:
					if *(elem.Delta) == *(value.Delta) {
						return true
					}
				case metric.MetricTypeGauge:
					if *(elem.Value) == *(value.Value) {
						return true
					}
				}

			}
		}
		return false
	}
	resMetrics, err := storage.GetAll(context.Background())
	assert.NoError(t, err)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.found, contains(resMetrics, test.want))
		})
	}
}

func TestGetByTypeAndName(t *testing.T) {
	tests := []struct {
		testName   string
		metricType string
		metricName string
		found      bool
		want       any
	}{
		{
			testName:   "positive counter",
			metricType: metric.MetricTypeCounter,
			metricName: "counterKey",
			found:      true,
			want:       int64(51),
		},
		{
			testName:   "positive gauge",
			metricType: metric.MetricTypeGauge,
			metricName: "gaugeKey",
			found:      true,
			want:       2.01,
		},
		{
			testName:   "positive gauge",
			metricType: metric.MetricTypeGauge,
			metricName: "gaugeKey2",
			found:      true,
			want:       1000.001,
		},
		{
			testName:   "negative counter",
			metricType: metric.MetricTypeCounter,
			metricName: "NotExistedKey",
			found:      false,
			want:       0,
		},
		{
			testName:   "negative gauge",
			metricType: metric.MetricTypeGauge,
			metricName: "NotExistedKey2",
			found:      false,
			want:       0,
		},
	}
	storage, err := getStorage(nil)
	if err != nil {
		assert.NoError(t, err)
	}
	Seed(storage)
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			resMetric, err := storage.GetByTypeAndName(context.Background(), test.metricType, test.metricName)
			if test.found {
				assert.NoError(t, err)
				switch test.metricType {
				case metric.MetricTypeGauge:
					assert.Equal(t, test.want, *(resMetric.Value))
				case metric.MetricTypeCounter:
					assert.Equal(t, test.want, *(resMetric.Delta))
				default:
					t.Errorf("unknown Mtype:%s", resMetric.MType)
				}
			} else {
				assert.ErrorIs(t, err, commonStorage.ErrMetricNotFound)
			}
		})
	}
}
func getStorage(configMap *config.Config) (*InMemory, error) {

	cfg := &config.Config{
		StoreInterval:   1000,
		FileStoragePath: `mytestfile.txt`,
		Restore:         false,
	}
	if configMap != nil {
		cfg = configMap
	}
	fileStorage := filestorage.NewStorage(cfg.FileStoragePath)
	storage, err := New(fileStorage, cfg)
	return storage, err
}
