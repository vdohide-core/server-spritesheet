package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// FileMetadata holds embedded metadata for a File.
type FileMetadata struct {
	Description *string     `bson:"description,omitempty" json:"description,omitempty"`
	Views       interface{} `bson:"views,omitempty" json:"views,omitempty"`
	Duration    *float64    `bson:"duration,omitempty" json:"duration,omitempty"`
	Highest     *int        `bson:"highest,omitempty" json:"highest,omitempty"`
	MimeType    *string     `bson:"mimeType,omitempty" json:"mimeType,omitempty"`
	Size        interface{} `bson:"size,omitempty" json:"size,omitempty"`
	TrashedAt   *time.Time  `bson:"trashedAt,omitempty" json:"trashedAt,omitempty"`
	TrashedBy   *string     `bson:"trashedBy,omitempty" json:"trashedBy,omitempty"`
	DeletedAt   *time.Time  `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	DeletedBy   *string     `bson:"deletedBy,omitempty" json:"deletedBy,omitempty"`
	Source      *string     `bson:"source,omitempty" json:"source,omitempty"`
	SourceType  *string     `bson:"sourceType,omitempty" json:"sourceType,omitempty"`
	SourceHash  *string     `bson:"sourceHash,omitempty" json:"sourceHash,omitempty"`
	Playlist    *string     `bson:"playlist,omitempty" json:"playlist,omitempty"`
}


// File represents a file/folder/space record.
// Collection: "files" | _id: String (UUID)
//
// Mongoose equivalent:
//
//	_id:       { type: String, required: true, default: uuidv4 }
//	slug:      { type: String, unique: true, default: () => randomString(11) }
//	status:    { type: String, enum: FileStatus, default: "waiting" }
//	type:      { type: String, enum: FileType, default: "video" }
//	parentId:  { type: String, ref: "File", index: true }
//	spaceId:   { type: String, ref: "File", index: true }
//	timestamps: true
type File struct {
	ID         string        `bson:"_id" json:"id" goose:"required,default:uuid"`
	Status     string        `bson:"status" json:"status" goose:"default:waiting"`
	Type       string        `bson:"type" json:"type" goose:"default:video"`
	Name       string        `bson:"name" json:"name" goose:"required"`
	CreatorID  *string       `bson:"creatorId,omitempty" json:"creatorId,omitempty" goose:"index"`
	ParentID   *string       `bson:"parentId,omitempty" json:"parentId,omitempty" goose:"ref:files,index"`
	SpaceID    *string       `bson:"spaceId,omitempty" json:"spaceId,omitempty" goose:"ref:workspaces,index"`
	Slug       string        `bson:"slug" json:"slug" goose:"unique,default:random(11),index"`
	ClonedFrom *string       `bson:"clonedFrom,omitempty" json:"clonedFrom,omitempty" goose:"ref:files"`
	Metadata   *FileMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt  time.Time     `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt  time.Time     `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// FileModel is the goose model for the "files" collection.
var FileModel = goose.NewModel[File]("files")
