package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// Verification represents a verification token.
// Collection: "verification" | _id: String
type Verification struct {
	ID         string    `bson:"_id" json:"id" goose:"required,default:uuid"`
	Identifier string    `bson:"identifier" json:"identifier"`
	Value      string    `bson:"value" json:"value"`
	ExpiresAt  time.Time `bson:"expiresAt" json:"expiresAt"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt  time.Time `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// VerificationModel is the goose model for the "verification" collection.
var VerificationModel = goose.NewModel[Verification]("verification")
