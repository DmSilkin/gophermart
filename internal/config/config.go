package config

import (
	"flag"
	"os"
	"time"

	"github.com/caarlos0/env"
)

type ServerConfig struct {
	HTTPAddress    string        `env:"RUN_ADDRESS" envDefault:"localhost:18080"`
	DatabaseURI    string        `env:"DATABASE_URI" envDefault:"postgres://postgres:0000@localhost:5432/gophermart"`
	AccrualAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8080"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" envDefault:"2s"`
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
	pollIntervalPtr := flag.Duration("i", cfg.PollInterval, "PollInterval for accrual -i=<filename>")

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

	if _, ok := os.LookupEnv("POLL_INTERVAL"); !ok {
		cfg.PollInterval = *pollIntervalPtr
	}

	return cfg, nil
}
