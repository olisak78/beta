package models

import (
	"time"

	"github.com/google/uuid"
)

type Token struct {
	UserUUID  uuid.UUID `json:"user_uuid" gorm:"type:uuid;primaryKey;not null"`
	Provider  string    `json:"provider" gorm:"size:50;primaryKey;not null"`
	Token     string    `json:"token" gorm:"not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
}

func (Token) TableName() string {
	return "tokens"
}
