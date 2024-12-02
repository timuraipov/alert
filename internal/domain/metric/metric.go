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
