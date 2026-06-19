package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// Ingest represents an upload/ingest record tracking file ingestion.
// Collection: "ingests" | _id: String (UUID)
type Ingest struct {
	ID         string    `bson:"_id" json:"id" goose:"required,default:uuid"`
	FileID     *string   `bson:"fileId,omitempty" json:"fileId,omitempty" goose:"ref:files,index"`
	StorageID  *string   `bson:"storageId,omitempty" json:"storageId,omitempty" goose:"ref:storages,index"`
	FileName   string    `bson:"fileName" json:"fileName"`
	Status     string    `bson:"status" json:"status" goose:"default:uploading"` // uploading, completed, failed
	Size       int64     `bson:"size" json:"size"`
	MimeType   *string   `bson:"mimeType,omitempty" json:"mimeType,omitempty"`
	Path       *string   `bson:"path,omitempty" json:"path,omitempty"`
	UploadedBy *string   `bson:"uploadedBy,omitempty" json:"uploadedBy,omitempty" goose:"ref:user"`
	SourceType string     `bson:"sourceType" json:"sourceType" goose:"default:upload"`
	DeletedAt  *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt  time.Time `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// IngestModel is the goose model for the "ingests" collection.
var IngestModel = goose.NewModel[Ingest]("ingests")
