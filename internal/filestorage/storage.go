package filestorage

import (
	"os"

	"github.com/timuraipov/alert/internal/logger"
	"go.uber.org/zap"
)

type Storage struct {
	fileName string
}

func NewStorage(filename string) *Storage {
	return &Storage{fileName: filename}
}
func (s *Storage) Write(data []byte) error {
	logger.Log.Debug("write data to disk", zap.String("data body", string(data)))

	return os.WriteFile(s.fileName, data, 0666)
}
func (s *Storage) Read() ([]byte, error) {
	data, err := os.ReadFile(s.fileName)
	return data, err
}
