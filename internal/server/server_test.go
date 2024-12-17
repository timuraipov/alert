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
	testCases := []struct {
		name         string
		path         string
		method       string
		expectedCode int
	}{
		{
			name:         "positive Counter",
			path:         "/update/counter/PollCount/100",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		// {
		// 	name:         "negative method GET",
		// 	path:         "/update/counter/PollCount/100",
		// 	method:       http.MethodGet,
		// 	expectedCode: http.StatusForbidden,
		// },
		{
			name:         "positive Gauge",
			path:         "/update/gauge/Alloc/100.1",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		{
			name:         "negative Gauge url shorter then must be",
			path:         "/update/gauge/",
			method:       http.MethodPost,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "negative Gauge value has incorrect type",
			path:         "/update/gauge/Alloc/asf",
			method:       http.MethodPost,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "negative Gauge type has incorrect value",
			path:         "/update/someType/Alloc/asf",
			method:       http.MethodPost,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "negative case without type",
			path:         "/update/Alloc/asf",
			method:       http.MethodPost,
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
			resp, _ := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(nil))
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}
func TestUpdateJson(t *testing.T) {
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

func TestGetByTypeAndName(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		method       string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "positive get Alloc",
			path:         "/value/gauge/Alloc",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedBody: "100.23",
		},
		{
			name:         "positive get PollCount",
			path:         "/value/counter/PollCount",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedBody: "100",
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
	storage.DBCounter["PollCount"] = 100
	storage.DBGauge["Alloc"] = 100.23
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(nil))
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedBody, body)
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
		expectedJSON string
	}{
		{
			name:         "positive get All",
			path:         "/",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedJSON: `[{"type":"gauge","id":"Alloc","value":100.11},{"id":"PollCount","delta":105,"type":"counter"}]`,
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
			assert.JSONEq(t, tt.expectedJSON, jsonBody)
			resp.Body.Close()
		})
	}
}

func TestGetByTypeAndNameJson(t *testing.T) {

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
		expectedJSON  string
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
			expectedJSON: `{"ID":"PollCount","MType":"counter","Delta":105}`,
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
			expectedJSON: `{"ID":"Alloc","MType":"gauge","Value":100.11}`,
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
				assert.JSONEq(t, tt.expectedJSON, jsonBody) // на сколько хорошо проверять таким образом?
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
