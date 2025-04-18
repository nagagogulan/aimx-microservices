package model

import "time"

type Role struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Description string    `json:"description"`
}

func (Role) TableName() string {
	return "role"
}
