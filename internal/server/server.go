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
	ctx := context.Background()
	storage, _ := inmemory.New()
	postgresStorage := postgres.Init(cfg.DatabaseDSN)
	fileStorage := filestorage.NewStorage(cfg.FileStoragePath)

	metricsHandler := metrics.New(storage, fileStorage, cfg)
	healthHandler := health.New(postgresStorage)
	r := MetricsRouter(metricsHandler)
	r.Get("/ping", healthHandler.Ping(ctx))
	return &Server{r: r, metricsHandler: metricsHandler, cfg: cfg}
}
func MetricsRouter(handler *metrics.MetricHandler) chi.Router {
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
