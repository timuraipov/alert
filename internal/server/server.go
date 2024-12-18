package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/timuraipov/alert/internal/handlers/metrics"
	"github.com/timuraipov/alert/internal/middleware/gzip"
	"github.com/timuraipov/alert/internal/middleware/logger"
	"github.com/timuraipov/alert/internal/storage/inmemory"
)

func New() chi.Router {
	storage, _ := inmemory.New()
	metricsHandler := metrics.MetricHandler{
		Storage: storage,
	}
	r := MetricsRouter(metricsHandler)
	return r
}
func MetricsRouter(handler metrics.MetricHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(logger.WithLogging)
	r.Use(gzip.GzipMiddleware)
	r.Post("/update/", handler.UpdateJSON)
	r.Post("/update/{type}/{name}/{val}", handler.Update)
	r.Post("/value/", handler.GetByNameJSON)
	r.Get("/value/{type}/{name}", handler.GetByName)
	r.Get("/", handler.GetAll)
	return r
}
