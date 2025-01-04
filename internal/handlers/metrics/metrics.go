package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/storage"
	"go.uber.org/zap"
)

type MetricHandler struct {
	Storage storage.DBStorage
}

func New(storage storage.DBStorage) *MetricHandler {
	metricsHandler := &MetricHandler{
		Storage: storage,
	}
	return metricsHandler
}
func (mh *MetricHandler) Shutdown() error {
	return mh.Storage.Flush()
}

var (
	ErrMetricNameRequired = errors.New("metric name is required")
	ErrMetricTypeIsEnum   = errors.New("metric type should be gauge or counter")
)

func (mh *MetricHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.GetAll"
	metrics, err := mh.Storage.GetAll(r.Context())
	if err != nil {
		logger.Log.Error("failed to get metrics",
			zap.String("operation", op),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	responseData, err := json.Marshal(metrics)
	if err != nil {
		logger.Log.Error("failed to Marshal body",
			zap.String("operation", op),
			zap.Error(err),
		)
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(responseData))
}
func (mh *MetricHandler) GetByNameJSON(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.GetByName"
	var metrics metric.Metrics
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {

		logger.Log.Error("failed to read incoming message",
			zap.String("operation", op),
			zap.Error(err),
		)
	}
	logger.Log.Debug("get request body",
		zap.String("operation", op),
		zap.String("requestBody", buf.String()),
	)
	if err := json.Unmarshal(buf.Bytes(), &metrics); err != nil {
		logger.Log.Error("failed to Unmarshal body",
			zap.String("operation", op),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	val, err := mh.Storage.GetByTypeAndName(r.Context(), metrics.MType, metrics.ID)
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		metrics.Delta = val.Delta
		metrics.Value = val.Value
		responseBody, err := json.Marshal(metrics)
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
func (mh *MetricHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	metricType := chi.URLParam(r, "type")
	val, err := mh.Storage.GetByTypeAndName(r.Context(), metricType, name)
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		w.WriteHeader(http.StatusOK)
		if val.MType == metric.MetricTypeCounter {
			w.Write([]byte(fmt.Sprintf("%v", *val.Delta)))
		}
		if val.MType == metric.MetricTypeGauge {
			w.Write([]byte(fmt.Sprintf("%v", *val.Value)))
		}

	}

}

func (mh *MetricHandler) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.UpdateJSON"
	var myMetrics metric.Metrics
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {

		logger.Log.Error("failed to read incoming message",
			zap.String("operation", op),
			zap.Error(err),
		)
	}
	logger.Log.Debug("get request body",
		zap.String("operation", op),
		zap.String("requestBody", buf.String()),
	)
	if err := json.Unmarshal(buf.Bytes(), &myMetrics); err != nil {
		logger.Log.Error("failed to Unmarshal body",
			zap.String("operation", op),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = parseAndValidateJSON(myMetrics)
	if err != nil {
		if errors.Is(err, ErrMetricNameRequired) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	responseObj, err := mh.Storage.Save(r.Context(), myMetrics)
	if err != nil {
		logger.Log.Error(" exception",
			zap.String("operation", op),
			zap.Error(err))
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

func parseAndValidateJSON(metrics metric.Metrics) error {
	if metrics.MType == "" {
		return ErrMetricNameRequired
	}
	if !(metrics.MType == metric.MetricTypeCounter || metrics.MType == metric.MetricTypeGauge) {
		return ErrMetricTypeIsEnum
	}

	return nil
}
func (mh *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.Update"
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "val")
	metric, err := parseAndValidate(metricType, metricName, metricValue)
	if err != nil {
		if errors.Is(err, ErrMetricNameRequired) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err = mh.Storage.Save(r.Context(), *metric)

	if err != nil {
		logger.Log.Error(" exception",
			zap.String("operation", op),
			zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func parseAndValidate(metricType, metricName string, value string) (*metric.Metrics, error) {
	if metricType == "" {
		return nil, ErrMetricNameRequired
	}

	metricObj := &metric.Metrics{
		MType: metricType,
	}

	switch metricType {
	case metric.MetricTypeCounter:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			fmt.Println("int64 mismatch", "-", value, "-")
			return nil, ErrMetricTypeIsEnum
		}
		metricObj.Delta = &val
	case metric.MetricTypeGauge:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			fmt.Printf("float64 mismatch")
			return nil, ErrMetricTypeIsEnum
		}
		metricObj.Value = &val
	default:
		return nil, ErrMetricTypeIsEnum
	}

	metricObj.ID = metricName
	return metricObj, nil
}
func (mh *MetricHandler) UpdateJSONBatch(w http.ResponseWriter, r *http.Request) {
	op := "handlers.metrics.UpdateJSONBatch"
	var myMetrics []metric.Metrics
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		logger.Log.Error("failed to read incoming message",
			zap.String("operation", op),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logger.Log.Debug("get request body",
		zap.String("operation", op),
		zap.String("requestBody", buf.String()),
	)
	if err := json.Unmarshal(buf.Bytes(), &myMetrics); err != nil {
		logger.Log.Error("failed to Unmarshal body",
			zap.String("operation", op),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, m := range myMetrics {
		err = parseAndValidateJSON(m)
		if err != nil {
			if errors.Is(err, ErrMetricNameRequired) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	err = mh.Storage.SaveBatch(r.Context(), myMetrics)
	if err != nil {
		logger.Log.Error("exception",
			zap.String("operation", op),
			zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
