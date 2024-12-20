package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	FlagRunAddr  string `env:"ADDRESS"`
	FlagLogLevel string `env:"LOG_LEVEL"`
}

func MustLoad() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.FlagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.FlagLogLevel, "l", "info", "setup flagLogLevel")
	flag.Parse()
	err := env.Parse(cfg)

	return cfg, err
}
