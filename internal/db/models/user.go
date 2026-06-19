package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// User represents a user account.
// Collection: "user" | _id: String (UUID)
// Note: Created by better-auth, no goose defaults needed.
type User struct {
	ID               string    `bson:"_id" json:"id"`
	Name             string    `bson:"name" json:"name"`
	Email            string    `bson:"email" json:"email"`
	EmailVerified    bool      `bson:"emailVerified" json:"emailVerified"`
	TwoFactorEnabled bool      `bson:"twoFactorEnabled" json:"twoFactorEnabled"`
	Role             string    `bson:"role" json:"role"`   // user, admin, super_admin, developer
	Image            *string   `bson:"image" json:"image"` // nullable
	CreatedAt        time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time `bson:"updatedAt" json:"updatedAt"`
}

// UserModel is the goose model for the "user" collection.
var UserModel = goose.NewModel[User]("user")

// UserMetadata represents user-specific key-value metadata.
// Collection: "user_metadata" | _id: String (UUID)
type UserMetadata struct {
	ID        string      `bson:"_id" json:"id"`
	UserID    string      `bson:"userId" json:"userId"`
	Name      string      `bson:"name" json:"name"`
	Value     interface{} `bson:"value" json:"value"`
	CreatedAt time.Time   `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time   `bson:"updatedAt" json:"updatedAt"`
}

// UserMetadataModel is the goose model for the "user_metadata" collection.
var UserMetadataModel = goose.NewModel[UserMetadata]("user_metadata")
