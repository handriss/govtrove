package models

import "time"

type AccountRequest struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	RequestType string    `json:"request_type"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateAccountRequestInput struct {
	RequestType string `json:"request_type"`
}
