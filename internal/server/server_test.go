package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			resp, _ := testRequest(t, ts, tt.method, tt.path)
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}
func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
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
		expectedBody string
	}{
		{
			name:         "positive get All",
			path:         "",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedBody: `Alloc = 100.23
PollCount = 100
`},
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
			resp, body := testRequest(t, ts, tt.method, tt.path)
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.NotEmpty(t, body)
			assert.Equal(t, tt.expectedBody, body)
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
			fmt.Println("host", tt.path)
			resp, body := testRequest(t, ts, tt.method, tt.path)
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedBody, body)
		})
	}
}
