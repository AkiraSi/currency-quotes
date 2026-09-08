package models

import (
	"time"

	"currency-quotes/common/currencies"
)

type Rate struct {
	Code      currencies.Currency
	Nominal   int64
	Value     int64
	UpdatedAt time.Time
}

type LatestResponse struct {
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PushRequest struct {
	Code string `json:"code"`
}

type PushResponse struct {
	UpdateID string `json:"update_id"`
}

type UpdateResponse struct {
	Status    string     `json:"status,omitempty"`
	Price     *float64   `json:"price,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}
