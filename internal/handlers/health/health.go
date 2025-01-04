package health

import (
	"net/http"

	"github.com/timuraipov/alert/internal/logger"
	"github.com/timuraipov/alert/internal/storage"
	"go.uber.org/zap"
)

type Health struct {
	DB storage.DBHealthStorage
}

func New(db storage.DBHealthStorage) *Health {
	return &Health{DB: db}
}
func (h *Health) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.DB.Ping(r.Context()); err != nil {
		logger.Log.Error("db error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}
