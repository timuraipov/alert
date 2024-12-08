package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/timuraipov/alert/internal/domain/metric"
)

type MetricStorage interface {
	Save(metric metric.Metric) error
	GetAll() map[string]interface{}
	GetByTypeAndName(metricType, metricName string) (interface{}, bool)
}
type MetricHandler struct {
	Storage MetricStorage
}

var (
	ErrMetricNameRequired = errors.New("metric name is required")
	ErrMetricTypeIsEnum   = errors.New("metric type should be gauge or counter")
)

func (mh *MetricHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	metrics := mh.Storage.GetAll()
	result := ""
	for key, val := range metrics {
		result += fmt.Sprintf("%s = %v", key, val) + "\n"
	}
	w.Write([]byte(result))
}
func (mh *MetricHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	metricType := chi.URLParam(r, "type")
	val, ok := mh.Storage.GetByTypeAndName(metricType, name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%v", val)))
	}

}

func (mh *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "val")
	fmt.Println("type, name, val", metricType, metricName, metricValue)
	fmt.Println("host", r.Host, "path", r.URL.Path)
	metric, err := parseAndValidate(metricType, metricName, metricValue)
	if err != nil {
		fmt.Print("not validated")
		if errors.Is(err, ErrMetricNameRequired) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = mh.Storage.Save(*metric)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
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
func parseAndValidate(metricType, metricName string, value string) (*metric.Metric, error) {

	if !(metricType == metric.MetricTypeCounter || metricType == metric.MetricTypeGauge) {
		fmt.Println("type mismatch")

		return nil, ErrMetricTypeIsEnum
	}
	metricObj := &metric.Metric{
		Type: metricType,
	}
	if metricType == metric.MetricTypeCounter {
		fmt.Println()
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			fmt.Println("int64 mismatch", "-", value, "-")
			return nil, ErrMetricTypeIsEnum
		}
		metricObj.ValueCounter = val
	}
	if metricType == metric.MetricTypeGauge {

		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			fmt.Printf("float64 mismatch")
			return nil, ErrMetricTypeIsEnum
		}
		metricObj.ValueGauge = val
	}
	metricObj.Name = metricName
	return metricObj, nil
}
