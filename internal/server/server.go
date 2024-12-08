package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/timuraipov/alert/internal/handlers/metrics"
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
	r.Use(middleware.Logger)
	r.Post("/update/{type}/{name}/{val}", handler.Update)
	r.Get("/value/{type}/{name}", handler.GetByName)
	r.Get("/", handler.GetAll)
	return r
}
