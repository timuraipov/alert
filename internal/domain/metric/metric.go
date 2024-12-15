package metric

const (
	MetricTypeGauge   = "gauge"
	MetricTypeCounter = "counter"
)

type Metric struct {
	Type         string
	Name         string
	ValueCounter int64
	ValueGauge   float64
}
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}
