package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timuraipov/alert/internal/storage/inmemory"
)

func TestUpdate(t *testing.T) {
	host := "http://localhost:8080"
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
		{
			name:         "negative method GET",
			path:         "/update/counter/PollCount/100",
			method:       http.MethodGet,
			expectedCode: http.StatusForbidden,
		},
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
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			saver, err := inmemory.New()
			if err != nil {
				panic(err)
			}
			request := httptest.NewRequest(tt.method, host+tt.path, nil)
			w := httptest.NewRecorder()
			handler := New(saver)
			handler(w, request)
			res := w.Result()
			assert.Equal(t, tt.expectedCode, res.StatusCode)
			defer res.Body.Close()
		})
	}
}
