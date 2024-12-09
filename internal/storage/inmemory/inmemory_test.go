package inmemory

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timuraipov/alert/internal/domain/metric"
)

func TestSaveGauge(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		metrics    []metric.Metric
		want       float64
		metricType string
	}{
		{
			name: "positive Gauge",
			err:  nil,
			metrics: []metric.Metric{
				{
					Type:       metric.MetricTypeGauge,
					Name:       "someName",
					ValueGauge: 1.009,
				},
			},
			want: 1.009,
		},
		{
			name: "positive Counter",
			err:  nil,
			metrics: []metric.Metric{
				{
					Type:       metric.MetricTypeGauge,
					Name:       "someName",
					ValueGauge: 1.009,
				},
				{
					Type:       metric.MetricTypeGauge,
					Name:       "someName",
					ValueGauge: 12.009,
				},
			},
			want: 12.009,
		},
	}

	for _, test := range tests {
		saver, err := New()
		assert.NoError(t, err)
		t.Run(test.name, func(t *testing.T) {
			for _, metric := range test.metrics {
				err := saver.Save(metric)
				assert.NoError(t, err)
			}

			assert.Equal(t, test.want, saver.DBGauge[test.metrics[0].Name])
		})
	}
}

func TestSaveCounter(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		metrics    []metric.Metric
		want       int64
		metricType string
	}{
		{
			name: "positive Counter 1 value",
			err:  nil,
			metrics: []metric.Metric{
				{
					Type:         metric.MetricTypeCounter,
					Name:         "someName",
					ValueCounter: 50,
				},
			},
			want: 50,
		},
		{
			name: "positive Counter multi value",
			err:  nil,
			metrics: []metric.Metric{
				{
					Type:         metric.MetricTypeCounter,
					Name:         "someName",
					ValueCounter: 1,
				},
				{
					Type:         metric.MetricTypeCounter,
					Name:         "someName",
					ValueCounter: 2,
				},
				{
					Type:         metric.MetricTypeCounter,
					Name:         "someName",
					ValueCounter: 1000,
				},
			},
			want: 1003,
		},
	}

	for _, test := range tests {
		saver, err := New()
		assert.NoError(t, err)
		t.Run(test.name, func(t *testing.T) {
			for _, metric := range test.metrics {
				err := saver.Save(metric)
				assert.NoError(t, err)
			}

			assert.Equal(t, test.want, saver.DBCounter[test.metrics[0].Name])
		})
	}
}
func Seed(storage *InMemory) {
	seeds := []metric.Metric{
		{
			Type:         metric.MetricTypeCounter,
			Name:         "counterKey",
			ValueCounter: 50,
		},
		{
			Type:         metric.MetricTypeCounter,
			Name:         "counterKey",
			ValueCounter: 1,
		},

		{
			Type:       metric.MetricTypeGauge,
			Name:       "gaugeKey",
			ValueGauge: 2.01,
		},
		{
			Type:       metric.MetricTypeGauge,
			Name:       "gaugeKey2",
			ValueGauge: 1000.001,
		},
	}
	for _, seed := range seeds {
		storage.Save(seed)
	}
}
func TestGetAll(t *testing.T) {
	tests := []struct {
		name string
		want map[string]interface{}
	}{
		{
			name: "positive test",
			want: map[string]interface{}{
				"counterKey": 51,
				"gaugeKey":   2.01,
				"gaugeKey2":  1000.001,
			},
		},
	}
	storage, err := New()
	if err != nil {
		fmt.Print(err)
	}
	Seed(storage)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, fmt.Sprint(test.want), fmt.Sprint(storage.GetAll()))
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
	storage, err := New()
	if err != nil {
		assert.NoError(t, err)
	}
	Seed(storage)
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			res, found := storage.GetByTypeAndName(test.metricType, test.metricName)
			assert.Equal(t, test.found, found)
			if test.found {
				assert.Equal(t, test.want, res)
			}
		})
	}
}
