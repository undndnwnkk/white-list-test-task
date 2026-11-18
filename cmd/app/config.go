package main

import (
	"errors"
	"os"
)

const defaultHTTPAddr = ":8080"

type config struct {
	DatabaseDSN string
	HTTPAddr    string
}

func loadConfig() (config, error) {
	cfg := config{
		DatabaseDSN: os.Getenv("DB_DSN"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
	}
	if cfg.DatabaseDSN == "" {
		return config{}, errors.New("DB_DSN environment variable is required")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = defaultHTTPAddr
	}
	return cfg, nil
}
