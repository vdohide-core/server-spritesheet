package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// Session represents a user session (better-auth).
// Collection: "session" | _id: String
// Note: Created by better-auth, no goose defaults needed.
type Session struct {
	ID        string    `bson:"_id" json:"id"`
	UserID    string    `bson:"userId" json:"userId"`
	Token     string    `bson:"token" json:"token"`
	ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt"`
	IPAddress *string   `bson:"ipAddress,omitempty" json:"ipAddress,omitempty"`
	UserAgent *string   `bson:"userAgent,omitempty" json:"userAgent,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// SessionModel is the goose model for the "session" collection.
var SessionModel = goose.NewModel[Session]("session")
