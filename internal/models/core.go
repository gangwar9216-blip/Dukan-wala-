package models

import "time"

type HealthcareCard struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	CardNumber string    `json:"card_number"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type Plan struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	PricePaise int64  `json:"price_paise"`
	Status     string `json:"status"`
}

type AssistanceRecord struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Requirement string    `json:"requirement"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Notification struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Type      string     `json:"type"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type MedicalHelpStatus struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Details   string    `json:"details"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
