package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// MediaMetadata holds embedded metadata for a Media record.
// Matches: mediaMetadataSchema (TS)
type MediaMetadata struct {
	Size      interface{} `bson:"size,omitempty" json:"size,omitempty"`
	Width     int         `bson:"width" json:"width"`
	Height    int         `bson:"height" json:"height"`
	Duration  float64     `bson:"duration" json:"duration"`
	DirectURL *string     `bson:"directUrl,omitempty" json:"directUrl,omitempty"`
}

// PrewarmData holds cache prewarm statistics.
type PrewarmData struct {
	Total   int `bson:"total" json:"total"`
	Hit     int `bson:"hit" json:"hit"`
	Miss    int `bson:"miss" json:"miss"`
	Expired int `bson:"expired" json:"expired"`
	Failed  int `bson:"failed" json:"failed"`
}

// PrewarmEntry holds a single prewarm entry per CDN/region.
type PrewarmEntry struct {
	Data      *PrewarmData `bson:"data,omitempty" json:"data,omitempty"`
	PrewarmAt *time.Time   `bson:"prewarmAt,omitempty" json:"prewarmAt,omitempty"`
}

// Media represents a processed media record (video, subtitle, thumbnail, etc.).
// Collection: "medias" | _id: String (UUID)
//
// Mongoose equivalent:
//
//	_id:        { type: String, required: true, default: uuidv4 }
//	type:       { type: String, enum: MediaType, default: MediaType.VIDEO }
//	slug:       { type: String, unique: true, default: () => randomString(11) }
//	resolution: { type: String, index: true }
//	storageId:  { type: String, index: true, ref: "Storage" }
//	sourceHash: { type: String, index: true }
//	fileId:     { type: String, ref: "File", index: true }
//	clonedFrom: { type: String, ref: "File", index: true }
//	timestamps: true
type Media struct {
	ID         string                  `bson:"_id" json:"id" goose:"required,default:uuid"`
	Type       string                  `bson:"type" json:"type" goose:"default:video"`
	FileName   *string                 `bson:"fileName,omitempty" json:"fileName,omitempty"`
	MimeType   *string                 `bson:"mimeType,omitempty" json:"mimeType,omitempty"`
	Resolution *string                 `bson:"resolution,omitempty" json:"resolution,omitempty" goose:"index"`
	StorageID  *string                 `bson:"storageId,omitempty" json:"storageId,omitempty" goose:"ref:storages,index"`
	Slug       string                  `bson:"slug" json:"slug" goose:"unique,default:random(11)"`
	Path       *string                 `bson:"path,omitempty" json:"path,omitempty"`
	SourceHash *string                 `bson:"sourceHash,omitempty" json:"sourceHash,omitempty" goose:"index"`
	FileID     *string                 `bson:"fileId,omitempty" json:"fileId,omitempty" goose:"ref:files,index"`
	ClonedFrom *string                 `bson:"clonedFrom,omitempty" json:"clonedFrom,omitempty" goose:"ref:files,index"`
	Metadata   *MediaMetadata          `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Prewarm    map[string]PrewarmEntry `bson:"prewarm,omitempty" json:"prewarm,omitempty"`
	DeletedAt  *time.Time              `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	CreatedAt  time.Time               `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt  time.Time               `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// MediaModel is the goose model for the "medias" collection.
var MediaModel = goose.NewModel[Media]("medias")
