package model

import "time"

type Withdrawal struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Destination    string    `json:"destination"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
}

type LedgerEntry struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	WithdrawalID int64     `json:"withdrawal_id"`
	Type         string    `json:"type"`
	Amount       float64   `json:"amount"`
	BalanceAfter float64   `json:"balance_after"`
	CreatedAt    time.Time `json:"created_at"`
}