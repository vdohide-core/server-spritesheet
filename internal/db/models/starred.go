package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// Starred represents a user's starred (bookmarked) file.
// Collection: "starreds" | _id: String (UUID)
type Starred struct {
	ID        string    `bson:"_id" json:"id" goose:"required,default:uuid"`
	UserID    string    `bson:"userId" json:"userId" goose:"ref:user,index"`
	FileID    string    `bson:"fileId" json:"fileId" goose:"ref:files,index"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// StarredModel is the goose model for the "starreds" collection.
var StarredModel = goose.NewModel[Starred]("starreds")
