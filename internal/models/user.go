package models

import (
	"time"
)

type User struct {
	ID           uint      `gorm:"primaryKey"`
	Username     string    `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string    `gorm:"size:255;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
