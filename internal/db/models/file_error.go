package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// FileError tracks files that have permanently failed processing.
// Used to skip files that keep failing, preventing infinite retry loops.
// Collection: "file_errors" | _id: String (UUID)
type FileError struct {
	ID        string    `bson:"_id" json:"id" goose:"required,default:uuid"`
	FileID    string    `bson:"fileId" json:"fileId" goose:"ref:files,index"`
	ErrorType string    `bson:"errorType" json:"errorType"` // spritesheet
	Error     string    `bson:"error,omitempty" json:"error,omitempty"`
	Slug      string    `bson:"slug,omitempty" json:"slug,omitempty"`
	WorkerID  string    `bson:"workerId,omitempty" json:"workerId,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// FileErrorModel is the goose model for the "file_errors" collection.
var FileErrorModel = goose.NewModel[FileError]("file_errors")
