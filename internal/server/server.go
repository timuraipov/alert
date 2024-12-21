package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/filestorage"
	"github.com/timuraipov/alert/internal/handlers/metrics"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/middleware/gzip"
	middlewareLogger "github.com/timuraipov/alert/internal/middleware/logger"
	"github.com/timuraipov/alert/internal/storage/inmemory"
	"go.uber.org/zap"
)

func New(cfg *config.Config) chi.Router {
	storage, _ := inmemory.New()
	fileStorage, err := filestorage.NewStorage(cfg.FileStoragePath)
	if err != nil {
		logger.Log.Error("failed to create new FileStorage", zap.Error(err))
	}
	metricsHandler := metrics.MetricHandler{
		Storage:     storage,
		FileStorage: fileStorage,
		Congig:      cfg,
	}
	r := MetricsRouter(metricsHandler)
	return r
}
func MetricsRouter(handler metrics.MetricHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewareLogger.WithLogging)
	r.Use(gzip.GzipMiddleware)
	r.Post("/update/", handler.UpdateJSON)
	r.Post("/update/{type}/{name}/{val}", handler.Update)
	r.Post("/value/", handler.GetByNameJSON)
	r.Get("/value/{type}/{name}", handler.GetByName)
	r.Get("/", handler.GetAll)
	return r
}
