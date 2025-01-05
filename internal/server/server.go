package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/timuraipov/alert/internal/config"
	"github.com/timuraipov/alert/internal/filestorage"
	"github.com/timuraipov/alert/internal/handlers/health"
	"github.com/timuraipov/alert/internal/handlers/metrics"
	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/middleware/gzip"
	middlewareLogger "github.com/timuraipov/alert/internal/middleware/logger"
	"github.com/timuraipov/alert/internal/storage/inmemory"
	"github.com/timuraipov/alert/internal/storage/postgres"
	"go.uber.org/zap"
)

type Server struct {
	r              chi.Router
	metricsHandler *metrics.MetricHandler
	cfg            *config.Config
}

func New(cfg *config.Config) *Server {
	op := "server.New"
	var (
		metricsHandler *metrics.MetricHandler
		healthHandler  *health.Health
	)
	ctx := context.Background()
	if len(cfg.DatabaseDSN) > 0 {
		postgresStorage, err := postgres.New(cfg.DatabaseDSN)
		if err != nil {
			logger.Log.Error("storage connection error",
				zap.String("operation", op),
				zap.Error(err),
			)
		}
		if err := postgresStorage.Bootstrap(ctx); err != nil {
			logger.Log.Fatal("error while bootstrap table", zap.Error(err))
		}

		metricsHandler = metrics.New(postgresStorage)
		healthHandler = health.New(postgresStorage)
	} else {
		fileStorage := filestorage.NewStorage(cfg.FileStoragePath)
		storage, _ := inmemory.New(fileStorage, cfg)
		metricsHandler = metrics.New(storage)
		healthHandler = health.New(storage)
	}

	r := MetricsRouter(metricsHandler)
	r.Get("/ping", healthHandler.Ping)
	return &Server{r: r, metricsHandler: metricsHandler, cfg: cfg}
}
func MetricsRouter(handler *metrics.MetricHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewareLogger.WithLogging)
	r.Use(gzip.GzipMiddleware)
	r.Post("/update/", handler.UpdateJSON)
	r.Post("/update/{type}/{name}/{val}", handler.Update)
	r.Post("/value/", handler.GetByNameJSON)
	r.Post("/updates/", handler.UpdateJSONBatch)
	r.Get("/value/{type}/{name}", handler.GetByName)
	r.Get("/", handler.GetAll)
	return r
}
func (s *Server) ListenAndServe() error {

	server := &http.Server{Addr: s.cfg.FlagRunAddr, Handler: s.r}

	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		signal := <-sig
		logger.Log.Info("get shutdown signal", zap.String("signal", signal.String()))

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancelFunc := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancelFunc()
		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()
		err := s.metricsHandler.Shutdown()
		if err != nil {
			logger.Log.Error("failed to shutdown metrics", zap.Error(err))

		}
		// Trigger graceful shutdown
		err = server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()
	// Run the server
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
	return nil
}
