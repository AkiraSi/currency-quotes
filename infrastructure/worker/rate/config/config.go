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
	if len(portValue) == 0 {
		return &Config{Port: 3000}, nil
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return nil, err
	}

	return &Config{
		Port: port,
	}, nil
}
