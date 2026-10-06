package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mobile    string    `json:"mobile"`
	Location  string    `json:"location"`
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type Partner struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	City     string `json:"city"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	Benefits string `json:"benefits"`
	Status   string `json:"status"`
}
type MedicalHelpRequest struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Details   string    `json:"details"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type Offer struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	StartsAt    *time.Time `json:"starts_at,omitempty"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	Status      string     `json:"status"`
}
