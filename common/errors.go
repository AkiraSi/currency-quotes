package common

import "errors"

const (
	MinPort = 1
	MaxPort = 65535
)

var (
	ErrHostRequired = errors.New("host is required")
	ErrPortInvalid  = errors.New("port must be between 1 and 65535")
)
