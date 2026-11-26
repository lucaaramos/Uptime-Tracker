package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string             `bson:"username" json:"username"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

type APIKey struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	Name       string             `bson:"name" json:"name"`
	Prefix     string             `bson:"prefix" json:"prefix"`
	KeyHash    string             `bson:"key_hash" json:"-"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	LastUsedAt *time.Time         `bson:"last_used_at,omitempty" json:"last_used_at,omitempty"`
}
