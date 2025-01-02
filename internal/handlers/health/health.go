package health

import (
	"context"
	"net/http"

	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/storage/postgres"
	"go.uber.org/zap"
)

type Health struct {
	DB *postgres.DB
}

func New(db *postgres.DB) *Health {
	return &Health{DB: db}
}
func (h *Health) Ping(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.DB.Ping(ctx); err != nil {
			logger.Log.Error("db error", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
	}

}
