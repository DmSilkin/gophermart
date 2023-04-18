package config

import (
	"flag"
	"os"

	"github.com/caarlos0/env"
)

type ServerConfig struct {
	HTTPAddress    string `env:"RUN_ADDRESS" envDefault:"localhost:18080"`
	DatabaseURI    string `env:"DATABASE_URI" envDefault:"postgres://postgres:0000@localhost:5432/gophermart"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:""`
}

func NewServerConfig() (ServerConfig, error) {
	var cfg ServerConfig

	err := env.Parse(&cfg)

	if err != nil {
		return ServerConfig{}, err
	}

	httpAddressPtr := flag.String("a", cfg.HTTPAddress, "HTTP-server address in format: -a=<ip>:<port>")
	databaseURIPtr := flag.String("d", cfg.DatabaseURI, "StoreInterval in seconds -d=<Duration>")
	accrualAddressPtr := flag.String("r", cfg.AccrualAddress, "StoreFile for metrics -r=<filename>")

	flag.Parse()

	if _, ok := os.LookupEnv("RUN_ADDRESS"); !ok {
		cfg.HTTPAddress = *httpAddressPtr
	}

	if _, ok := os.LookupEnv("DATABASE_URI"); !ok {
		cfg.DatabaseURI = *databaseURIPtr
	}

	if _, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); !ok {
		cfg.AccrualAddress = *accrualAddressPtr
	}

	return cfg, nil
}
