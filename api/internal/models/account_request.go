package models

import "time"

type AccountRequest struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	RequestType string     `json:"request_type"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CompletedBy *string    `json:"completed_by,omitempty"`
	ExportS3Key *string    `json:"export_s3_key,omitempty"`
}

type AccountRequestWithUser struct {
	AccountRequest
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
}

type CreateAccountRequestInput struct {
	RequestType string `json:"request_type"`
}
