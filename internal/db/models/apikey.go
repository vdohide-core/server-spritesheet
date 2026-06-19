package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// ApiKey represents an API key for programmatic access.
// Collection: "api_keys" | _id: String (UUID)
type ApiKey struct {
	ID         string     `bson:"_id" json:"id" goose:"required,default:uuid"`
	Name       string     `bson:"name" json:"name" goose:"required"`
	KeyHash    string     `bson:"keyHash" json:"-" goose:"required,unique"`
	Prefix     string     `bson:"prefix" json:"prefix" goose:"required"`
	CreatorID  string     `bson:"creatorId" json:"creatorId" goose:"ref:user,index"`
	SpaceID    string     `bson:"spaceId" json:"spaceId" goose:"ref:workspaces,index"`
	LastUsedAt *time.Time `bson:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt  time.Time  `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// ApiKeyModel is the goose model for the "api_keys" collection.
var ApiKeyModel = goose.NewModel[ApiKey]("api_keys")
