package model

import "time"

type User struct {
	ID        int64     `json:"v_id"`
	Name      string    `json:"v_name"`
	Balance   float64   `json:"v_balance"`
	CreatedAt time.Time `json:"v_created_at"`
}