package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timuraipov/alert/internal/common"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/handlers/metrics"
	"github.com/timuraipov/alert/internal/storage/inmemory"
)

func TestUpdate(t *testing.T) {
	type expectedBody struct {
		id    string
		mType string
		delta *int64
		value *float64
	}
	testCases := []struct {
		name         string
		path         string
		method       string
		metric       metric.Metrics
		expectedBody metric.Metrics
		expectedCode int
	}{
		{
			name:   "positive Counter",
			path:   "/update",
			method: http.MethodPost,
			metric: metric.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: common.Pointer(int64(100)),
			},
			expectedBody: metric.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: common.Pointer(int64(100)),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:   "positive Counter case2",
			path:   "/update",
			method: http.MethodPost,
			metric: metric.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: common.Pointer(int64(100)),
			},
			expectedBody: metric.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: common.Pointer(int64(200)),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:   "positive Gauge",
			path:   "/update",
			method: http.MethodPost,
			metric: metric.Metrics{
				ID:    "Alloc",
				MType: metric.MetricTypeGauge,
				Value: common.Pointer(100.1),
			},
			expectedBody: metric.Metrics{
				ID:    "Alloc",
				MType: metric.MetricTypeGauge,
				Value: common.Pointer(100.1),
			},
			expectedCode: http.StatusOK,
		},
		{
			name:   "negative Unsupported Type",
			path:   "/update",
			method: http.MethodPost,
			metric: metric.Metrics{
				ID:    "Alloc",
				MType: "unsupported type",
				Value: common.Pointer(100.1),
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "negative Gauge mType has empty type",
			path:   "/update",
			method: http.MethodPost,
			metric: metric.Metrics{
				ID:    "Alloc",
				MType: "",
				Value: common.Pointer(100.1),
			},
			expectedCode: http.StatusNotFound,
		},
	}
	storage, err := inmemory.New()
	if err != nil {
		panic(err)
	}
	metricsHandler := metrics.MetricHandler{
		Storage: storage,
	}
	ts := httptest.NewServer(MetricsRouter(metricsHandler))
	defer ts.Close()
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.metric)
			assert.NoError(t, err)
			resp, actualMetric := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(requestBody))
			expectedMetricBytes, err := json.Marshal(tt.expectedBody)
			assert.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if tt.expectedCode == http.StatusOK {
				assert.JSONEq(t, string(expectedMetricBytes), actualMetric)
			}
		})
	}
}
func testRequest(t *testing.T, ts *httptest.Server, method string, path string, serializedBody *bytes.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, serializedBody)
	require.NoError(t, err)
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(respBody)
}

func TestAll(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		method       string
		expectedCode int
		expectedJson string
	}{
		{
			name:         "positive get All",
			path:         "/",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedJson: `[{"type":"gauge","id":"Alloc","value":100.11},{"id":"PollCount","delta":105,"type":"counter"}]`,
		},
	}
	storage, err := inmemory.New()
	if err != nil {
		panic(err)
	}
	metricsHandler := metrics.MetricHandler{
		Storage: storage,
	}
	ts := httptest.NewServer(MetricsRouter(metricsHandler))
	defer ts.Close()
	preSeed(storage)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resp, jsonBody := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(nil))
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.JSONEq(t, tt.expectedJson, jsonBody)
			resp.Body.Close()
		})
	}
}

func TestGetByTypeAndName(t *testing.T) {

	type metricReqRes struct {
		ID    string   `json:"ID"`
		MType string   `json:"MType"`
		Delta *int64   `json:"Delta,omitempty"`
		Value *float64 `json:"Value,omitempty"`
	}
	testCases := []struct {
		name          string
		path          string
		method        string
		expectedCode  int
		metricReqBody metricReqRes
		expectedJson  string
	}{
		{
			name:         "positive get PollCount type counter",
			path:         "/value",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
			metricReqBody: metricReqRes{
				ID:    "PollCount",
				MType: "counter",
			},
			expectedJson: `{"ID":"PollCount","MType":"counter","Delta":105}`,
		},
		{
			name:         "positive get Alloc type gauge",
			path:         "/value",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
			metricReqBody: metricReqRes{
				ID:    "Alloc",
				MType: "gauge",
			},
			expectedJson: `{"ID":"Alloc","MType":"gauge","Value":100.11}`,
		},
	}
	storage, err := inmemory.New()
	if err != nil {
		panic(err)
	}
	metricsHandler := metrics.MetricHandler{
		Storage: storage,
	}
	ts := httptest.NewServer(MetricsRouter(metricsHandler))
	defer ts.Close()
	preSeed(storage)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.metricReqBody)
			assert.NoError(t, err)
			resp, jsonBody := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(requestBody))
			resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if tt.expectedCode == http.StatusOK {
				assert.JSONEq(t, tt.expectedJson, jsonBody) // на сколько хорошо проверять таким образом?
			}
		})
	}
}
func preSeed(storage *inmemory.InMemory) {
	seeds := []metric.Metrics{
		{
			ID:    "PollCount",
			MType: "counter",
			Delta: common.Pointer(int64(100)),
		},
		{
			ID:    "PollCount",
			MType: "counter",
			Delta: common.Pointer(int64(5)),
		},
		{
			ID:    "Alloc",
			MType: "gauge",
			Value: common.Pointer(100.11),
		},
	}
	for _, seed := range seeds {
		storage.Save(seed)
	}
}
