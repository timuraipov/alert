package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	FlagRunAddr string `env:"ADDRESS"`
}

func MustLoad() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.FlagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.Parse()
	err := env.Parse(cfg)

	return cfg, err
}
