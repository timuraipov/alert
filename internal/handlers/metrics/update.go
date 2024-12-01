package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	. "github.com/timuraipov/alert/internal/domain/metric"
)

type MetricSaver interface {
	Save(metric Metric) error
}

var (
	ErrMetricNameRequired = errors.New("metric name is required")
	ErrMetricTypeIsEnum   = errors.New("metric type should be gauge or counter")
)

func New(saver MetricSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fmt.Println("host", r.Host, "path", r.URL.Path)
		path := r.URL.Path
		splittedPath := strings.Split(path[1:], "/")
		metric, err := parseAndValidate(splittedPath)
		if err != nil {
			if errors.Is(err, ErrMetricNameRequired) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = saver.Save(*metric)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// func Update(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		w.WriteHeader(http.StatusForbidden)
// 		return
// 	}
// 	fmt.Println("host", r.Host, "path", r.URL.Path)
// 	path := r.URL.Path
// 	splittedPath := strings.Split(path[1:], "/")
// 	err := validate(splittedPath)
// 	if err != nil {
// 		if errors.Is(err, ErrMetricNameRequired) {
// 			w.WriteHeader(http.StatusNotFound)
// 			return
// 		}
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// }
func parseAndValidate(splittedPath []string) (*Metric, error) {
	if len(splittedPath) != 4 {
		fmt.Println("len mismatch")
		return nil, ErrMetricNameRequired
	}
	metricType := splittedPath[1]
	if !(metricType == MetricTypeCounter || metricType == MetricTypeGauge) {
		fmt.Println("type mismatch")

		return nil, ErrMetricTypeIsEnum
	}
	metric := &Metric{
		Type: metricType,
	}
	if metricType == MetricTypeCounter {

		value, err := strconv.ParseInt(splittedPath[3], 10, 64)
		if err != nil {
			return nil, ErrMetricTypeIsEnum
		}
		metric.ValueCounter = value
	}
	if metricType == MetricTypeGauge {

		value, err := strconv.ParseFloat(splittedPath[3], 64)
		if err != nil {
			return nil, ErrMetricTypeIsEnum
		}
		metric.ValueGauge = value
	}
	metric.Name = splittedPath[2]
	return metric, nil
}
