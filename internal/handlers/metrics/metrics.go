package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/filestorage"
	"github.com/timuraipov/alert/internal/logger"
	"go.uber.org/zap"
)

type MetricStorage interface {
	Save(metric metric.Metrics) (metric.Metrics, error)
	GetAll() []metric.Metrics
	GetByTypeAndName(metricType, metricName string) (metric.Metrics, bool)
}
type MetricHandler struct {
	Storage         MetricStorage
	fileStorage     *filestorage.Storage
	config          *config.Config
	IsNeedSyncFlush bool
}

func New(storage MetricStorage, fileStorage *filestorage.Storage, cfg *config.Config) *MetricHandler {
	metricsHandler := &MetricHandler{
		Storage:     storage,
		fileStorage: fileStorage,
		config:      cfg,
	}
	metricsHandler.init()
	return metricsHandler
}
func (mh *MetricHandler) init() {
	if mh.config.Restore {
		mh.load()
	}
	if mh.config.StoreInterval == 0 {
		mh.IsNeedSyncFlush = true
	} else {
		tickerFlushToDisk := time.NewTicker(time.Duration(mh.config.StoreInterval) * time.Second)
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-tickerFlushToDisk.C:
					mh.flush()
				case <-done:
					tickerFlushToDisk.Stop()
					return
				}
			}
		}()
	}

	//---

}
func (mh *MetricHandler) Shutdown() error {
	return mh.flush()
}

func (mh *MetricHandler) load() error {
	metrics := make([]metric.Metrics, 0)
	data, err := mh.fileStorage.Read()
	if err != nil {
		logger.Log.Error("failed to load file", zap.Error(err), zap.String("data", string(data)))
	}
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return fmt.Errorf("failed to load file %w", err)
	}
	for _, parsedMetric := range metrics {
		_, err = mh.Storage.Save(parsedMetric)
		if err != nil {
			logger.Log.Error("failed to save metric", zap.Error(err))
		}
	}
	return nil
}

func (mh *MetricHandler) flush() error {
	logger.Log.Debug(
		"called flush to disk method", zap.String("op", "op"),
	)
	metrics := mh.Storage.GetAll()
	if len(metrics) > 0 {
		data, err := json.Marshal(metrics)
		if err != nil {
			return fmt.Errorf("failed to Marshal metrics %w", err)
		}
		err = mh.fileStorage.Write(data)
		if err != nil {
			return fmt.Errorf("failed write metrics to disk %w", err)
		}
	}
	return nil
}

var (
	ErrMetricNameRequired = errors.New("metric name is required")
	ErrMetricTypeIsEnum   = errors.New("metric type should be gauge or counter")
)

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
	val, ok := mh.Storage.GetByTypeAndName(metrics.MType, metrics.ID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
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
	val, ok := mh.Storage.GetByTypeAndName(metricType, name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
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
	if mh.IsNeedSyncFlush {
		err := mh.flush()
		if err != nil {
			logger.Log.Error("can't flush metrics on disk", zap.Error(err))
		}
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
	_, err = mh.Storage.Save(*metric)
	if mh.IsNeedSyncFlush {
		err := mh.flush()
		logger.Log.Error("can't flush metrics on disk", zap.Error(err))
	}
	if err != nil {
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
