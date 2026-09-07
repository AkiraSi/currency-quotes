package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port int
}

func NewConfig() (*Config, error) {
	portValue := os.Getenv("RATE_WORKER_PORT")
	if portValue == "" {
		return nil, nil
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return nil, err
	}

	return &Config{
		Port: port,
	}, nil
}
