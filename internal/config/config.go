package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	FlagRunAddr     string `env:"ADDRESS"`
	FlagLogLevel    string `env:"LOG_LEVEL"`
	StoreInterval   int64  `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func MustLoad() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.FlagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.FlagLogLevel, "l", "info", "setup flagLogLevel")
	flag.Int64Var(&cfg.StoreInterval, "i", 300, "time for flush to disk")
	flag.StringVar(&cfg.FileStoragePath, "f", "metrics_file.txt", "file name for flush to disk")
	flag.BoolVar(&cfg.Restore, "r", true, "flag to indicate if need to load metrics from file")
	flag.Parse()
	err := env.Parse(cfg)

	return cfg, err
}
