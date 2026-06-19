package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// Account represents an authentication account (better-auth).
// Collection: "account" | _id: String
// Note: Created by better-auth, no goose defaults needed.
type Account struct {
	ID                    string     `bson:"_id" json:"id"`
	UserID                string     `bson:"userId" json:"userId"`
	AccountID             string     `bson:"accountId" json:"accountId"`
	ProviderID            string     `bson:"providerId" json:"providerId"`
	AccessToken           *string    `bson:"accessToken,omitempty" json:"accessToken,omitempty"`
	RefreshToken          *string    `bson:"refreshToken,omitempty" json:"refreshToken,omitempty"`
	AccessTokenExpiresAt  *time.Time `bson:"accessTokenExpiresAt,omitempty" json:"accessTokenExpiresAt,omitempty"`
	RefreshTokenExpiresAt *time.Time `bson:"refreshTokenExpiresAt,omitempty" json:"refreshTokenExpiresAt,omitempty"`
	Scope                 *string    `bson:"scope,omitempty" json:"scope,omitempty"`
	IDToken               *string    `bson:"idToken,omitempty" json:"idToken,omitempty"`
	Password              *string    `bson:"password,omitempty" json:"-"`
	CreatedAt             time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time  `bson:"updatedAt" json:"updatedAt"`
}

// AccountModel is the goose model for the "account" collection.
var AccountModel = goose.NewModel[Account]("account")
