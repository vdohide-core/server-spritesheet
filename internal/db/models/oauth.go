package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// OAuth represents a Google Drive OAuth credential.
// Collection: "oauths" | _id: String (UUID)
type OAuth struct {
	ID           string      `bson:"_id" json:"id" goose:"required,default:uuid"`
	Enable       bool        `bson:"enable" json:"enable"`
	Email        string      `bson:"email" json:"email" goose:"index"`
	CreatorID    *string     `bson:"creatorId,omitempty" json:"creatorId,omitempty" goose:"ref:user,index"`
	SpaceID      *string     `bson:"spaceId,omitempty" json:"spaceId,omitempty" goose:"ref:workspaces,index"`
	ClientID     *string     `bson:"client_id,omitempty" json:"clientId,omitempty"`
	ClientSecret *string     `bson:"client_secret,omitempty" json:"-"`
	RefreshToken *string     `bson:"refresh_token,omitempty" json:"-"`
	Token        interface{} `bson:"token,omitempty" json:"-"`
	TokenAt      *time.Time  `bson:"tokenAt,omitempty" json:"tokenAt,omitempty"`
	CreatedAt    time.Time   `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt    time.Time   `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// OAuthModel is the goose model for the "oauths" collection.
var OAuthModel = goose.NewModel[OAuth]("oauths")
