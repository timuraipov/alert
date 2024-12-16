package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/logger"
	"go.uber.org/zap"
)

type MetricStorage interface {
	Save(metric metric.Metrics) (metric.Metrics, error)
	GetAll() []metric.Metrics
	GetByTypeAndName(metricType, metricName string) (metric.Metrics, bool)
}
type MetricHandler struct {
	Storage MetricStorage
}

var (
	ErrMetricNameRequired = errors.New("metric name is required")
	ErrMetricTypeIsEnum   = errors.New("metric type should be gauge or counter")
)

type metricResponse struct {
	ID    string   `json:"ID"`
	MType string   `json:"MType"`
	Delta *int64   `json:"Delta,omitempty"`
	Value *float64 `json:"Value,omitempty"`
}

func (mh *MetricHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.GetAll"

	metrics := mh.Storage.GetAll()
	responseData, err := json.Marshal(metrics)
	if err != nil {
		logger.Log.Error("failed to Marshal body",
			zap.String("operation", op),
			zap.Error(err),
		)
	}
	w.Write([]byte(responseData))
}
func (mh *MetricHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.GetByName"
	var myMetrics metricResponse
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {

		logger.Log.Error("failed to read incoming message",
			zap.String("operation", op),
			zap.Error(err),
		)
	}
	if err := json.Unmarshal(buf.Bytes(), &myMetrics); err != nil {
		logger.Log.Error("failed to Unmarshal body",
			zap.String("operation", op),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	val, ok := mh.Storage.GetByTypeAndName(myMetrics.MType, myMetrics.ID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		myMetrics.Delta = val.Delta
		myMetrics.Value = val.Value
		responseBody, err := json.Marshal(myMetrics)
		if err != nil {
			logger.Log.Error("failed to Marshal response",
				zap.String("operation", op),
				zap.Error(err),
			)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	}

}

func (mh *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.Update"
	var myMetrics metric.Metrics
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {

		logger.Log.Error("failed to read incoming message",
			zap.String("operation", op),
			zap.Error(err),
		)
	}
	if err := json.Unmarshal(buf.Bytes(), &myMetrics); err != nil {
		logger.Log.Error("failed to Unmarshal body",
			zap.String("operation", op),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = parseAndValidate(myMetrics)
	if err != nil {
		if errors.Is(err, ErrMetricNameRequired) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	responseObj, err := mh.Storage.Save(myMetrics)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	responseBody, err := json.Marshal(responseObj)
	if err != nil {
		logger.Log.Error("failed to Marshal response",
			zap.String("operation", op),
			zap.Error(err),
		)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseBody)
}

func parseAndValidate(metrics metric.Metrics) error {
	if metrics.MType == "" {
		return ErrMetricNameRequired
	}
	if !(metrics.MType == metric.MetricTypeCounter || metrics.MType == metric.MetricTypeGauge) {
		return ErrMetricTypeIsEnum
	}

	return nil
}
