package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timuraipov/alert/internal/common"
	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/domain/metric"
	"github.com/timuraipov/alert/internal/filestorage"
	"github.com/timuraipov/alert/internal/handlers/metrics"
	"github.com/timuraipov/alert/internal/storage"
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

	ts, _ := getServer()
	defer ts.Close()
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(nil), nil)
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}
func TestUpdateJson(t *testing.T) {

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
			path:   "/update/",
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
			path:   "/update/",
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
			path:   "/update/",
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
			path:   "/update/",
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
			path:   "/update/",
			method: http.MethodPost,
			metric: metric.Metrics{
				ID:    "Alloc",
				MType: "",
				Value: common.Pointer(100.1),
			},
			expectedCode: http.StatusNotFound,
		},
	}

	ts, _ := getServer()
	defer ts.Close()
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.metric)
			assert.NoError(t, err)
			resp, actualMetric := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(requestBody), nil)
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
			expectedBody: "100.11",
		},
		{
			name:         "positive get PollCount",
			path:         "/value/counter/PollCount",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedBody: "105",
		},
	}

	ts, mh := getServer()
	defer ts.Close()

	preSeed(mh.Storage)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(nil), nil)
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedBody, body)
		})
	}
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
			path:         "/?",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedJSON: `[{"type":"gauge","id":"Alloc","value":100.11},{"id":"PollCount","delta":105,"type":"counter"}]`,
		},
	}

	ts, mh := getServer()
	defer ts.Close()
	preSeed(mh.Storage)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resp, jsonBody := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(nil), nil)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.JSONEq(t, tt.expectedJSON, jsonBody)
			resp.Body.Close()
		})
	}
}

func TestGetByTypeAndNameJson(t *testing.T) {

	testCases := []struct {
		name          string
		path          string
		method        string
		expectedCode  int
		metricReqBody metric.Metrics
		expectedJSON  string
	}{
		{
			name:         "positive get PollCount type counter",
			path:         "/value/",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
			metricReqBody: metric.Metrics{
				ID:    "PollCount",
				MType: "counter",
			},
			expectedJSON: `{"id":"PollCount","type":"counter","delta":105}`,
		},
		{
			name:         "positive get Alloc type gauge",
			path:         "/value/",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
			metricReqBody: metric.Metrics{
				ID:    "Alloc",
				MType: "gauge",
			},
			expectedJSON: `{"id":"Alloc","type":"gauge","value":100.11}`,
		},
	}

	ts, mh := getServer()
	defer ts.Close()
	preSeed(mh.Storage)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.metricReqBody)
			assert.NoError(t, err)
			resp, jsonBody := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(requestBody), nil)
			resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			if tt.expectedCode == http.StatusOK {
				assert.JSONEq(t, tt.expectedJSON, jsonBody) // на сколько хорошо проверять таким образом?
			}
		})
	}
}
func TestUpdates(t *testing.T) {
	testCase := []metric.Metrics{
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
	tests := []struct {
		name         string
		testCase     []metric.Metrics
		expectedCode int
		method       string
		path         string
	}{
		{
			name:         "positive batch",
			testCase:     testCase,
			expectedCode: http.StatusOK,
			method:       http.MethodPost,
			path:         "/updates/",
		},
	}

	ts, _ := getServer()
	defer ts.Close()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.testCase)
			assert.NoError(t, err)
			resp, _ := testRequest(t, ts, tt.method, tt.path, bytes.NewReader(requestBody), nil)
			assert.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

		})
	}

}
func TestGetByTypeAndNameGZIP(t *testing.T) {
	requestBody := `{"id":"PollCount","type":"counter"}`
	successBody := `{"id":"PollCount","type":"counter","delta":105}`
	path := "/value/"
	ts, mh := getServer()
	defer ts.Close()
	preSeed(mh.Storage)

	t.Run("send ByTypeAndName gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		headers := make(map[string]string, 2)
		headers["Content-Encoding"] = "gzip"
		headers["Accept-Encoding"] = ""
		assert.NoError(t, err)
		resp, jsonBody := testRequest(t, ts, http.MethodPost, path, buf, &headers)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, resp.StatusCode, http.StatusOK)
		assert.JSONEq(t, successBody, jsonBody)
	})
	t.Run("send ByTypeAndName  accept gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		headers := make(map[string]string, 1)
		headers["Accept-Encoding"] = "gzip"
		resp, respBody := testRequest(t, ts, http.MethodPost, path, buf, &headers)
		zr, err := gzip.NewReader(strings.NewReader(respBody))
		resp.Body.Close()
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		b, err := io.ReadAll(zr)
		require.NoError(t, err)

		require.JSONEq(t, successBody, string(b))

	})
}
func preSeed(storage storage.DBStorage) {
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
		_, err := storage.Save(context.Background(), seed)
		if err != nil {
			panic(err)
		}
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method string, path string, serializedBody io.Reader, headers *map[string]string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, serializedBody)
	require.NoError(t, err)
	if headers != nil {
		for key, val := range *headers {
			req.Header.Set(key, val)
		}
	}

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(respBody)
}
func getServer() (*httptest.Server, *metrics.MetricHandler) {

	cfg := &config.Config{
		StoreInterval:   1000,
		FileStoragePath: `mytestfile.txt`,
		Restore:         false,
	}
	fileStorage := filestorage.NewStorage(cfg.FileStoragePath)
	storage, err := inmemory.New(fileStorage, cfg)
	if err != nil {
		panic(err)
	}
	metricsHandler := metrics.New(storage)
	return httptest.NewServer(MetricsRouter(metricsHandler)), metricsHandler
}
