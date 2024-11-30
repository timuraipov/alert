package inmemory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timuraipov/alert/internal/handlers/metrics"
)

func TestSaveGauge(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		metrics    []metrics.Metric
		want       float64
		metricType string
	}{
		{
			name: "positive Gauge",
			err:  nil,
			metrics: []metrics.Metric{
				{
					Type:       metrics.MetricTypeGauge,
					Name:       "someName",
					ValueGauge: 1.009,
				},
			},
			want: 1.009,
		},
		{
			name: "positive Counter",
			err:  nil,
			metrics: []metrics.Metric{
				{
					Type:       metrics.MetricTypeGauge,
					Name:       "someName",
					ValueGauge: 1.009,
				},
				{
					Type:       metrics.MetricTypeGauge,
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
		metrics    []metrics.Metric
		want       int64
		metricType string
	}{
		{
			name: "positive Counter 1 value",
			err:  nil,
			metrics: []metrics.Metric{
				{
					Type:         metrics.MetricTypeCounter,
					Name:         "someName",
					ValueCounter: 50,
				},
			},
			want: 50,
		},
		{
			name: "positive Counter multi value",
			err:  nil,
			metrics: []metrics.Metric{
				{
					Type:         metrics.MetricTypeCounter,
					Name:         "someName",
					ValueCounter: 1,
				},
				{
					Type:         metrics.MetricTypeCounter,
					Name:         "someName",
					ValueCounter: 2,
				},
				{
					Type:         metrics.MetricTypeCounter,
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
