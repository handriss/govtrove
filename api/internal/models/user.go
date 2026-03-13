package models

import "time"

type User struct {
	ID        int       `json:"id"`
	WorkOSID  string    `json:"workos_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Plan               string     `json:"plan"`
	FreeForever        bool       `json:"free_forever"`
	StripeCustomerID   *string    `json:"-"`
	SubscriptionID     *string    `json:"-"`
	SubscriptionStatus *string    `json:"subscription_status,omitempty"`
	CancelAtPeriodEnd  bool       `json:"cancel_at_period_end"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	GiftExpiresAt      *time.Time `json:"gift_expires_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UpsertUserInput struct {
	WorkOSID  string `json:"workos_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
