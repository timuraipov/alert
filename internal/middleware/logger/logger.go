package logger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/timuraipov/alert/internal/logger"
)

type (
	requestLog struct {
		URI      string
		method   string
		duration time.Duration
	}
	responseLog struct {
		status int
		size   int
	}
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}
func WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r) // внедряем реализацию http.ResponseWriter

		duration := time.Since(start)

		logger.Log.Sugar().Info(
			"request", fmt.Sprintf(`{"URI":%v,"method":"%v","duration":%v}`, r.RequestURI, r.Method, duration),
			"response", fmt.Sprintf(`{"status":%v,"size":%v}`, responseData.status, responseData.size),
		)
	}
	return http.HandlerFunc(logFn)
}
