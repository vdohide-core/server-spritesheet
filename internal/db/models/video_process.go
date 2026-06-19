package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// Process statuses
const (
	ProcessStatusPending    = "pending"
	ProcessStatusProcessing = "processing"
	ProcessStatusCompleted  = "completed"
	ProcessStatusFailed     = "failed"
	ProcessStatusCancelled  = "cancelled"
)

// Step statuses
const (
	StepStatusPending    = "pending"
	StepStatusProcessing = "processing"
	StepStatusCompleted  = "completed"
	StepStatusFailed     = "failed"
)

// StepProgress holds progress info for a processing step.
type StepProgress struct {
	Status    *string    `bson:"status,omitempty" json:"status,omitempty"`
	Percent   *float64   `bson:"percent,omitempty" json:"percent,omitempty"`
	Current   *int       `bson:"current,omitempty" json:"current,omitempty"`
	Total     *int       `bson:"total,omitempty" json:"total,omitempty"`
	Speed     *string    `bson:"speed,omitempty" json:"speed,omitempty"`
	StartedAt *time.Time `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	EndedAt   *time.Time `bson:"endedAt,omitempty" json:"endedAt,omitempty"`
}

// VideoProcess represents a video processing job.
// Collection: "video_process" | _id: String
type VideoProcess struct {
	ID             string      `bson:"_id" json:"id" goose:"required,default:uuid"`
	SpaceID        *string     `bson:"spaceId,omitempty" json:"spaceId,omitempty" goose:"ref:workspaces,index"`
	FileID         *string     `bson:"fileId,omitempty" json:"fileId,omitempty" goose:"ref:files,index"`
	Slug           *string     `bson:"slug,omitempty" json:"slug,omitempty" goose:"index"`
	WorkerID       *string     `bson:"workerId,omitempty" json:"workerId,omitempty" goose:"index"`
	Status         *string     `bson:"status,omitempty" json:"status,omitempty" goose:"index"`
	OverallPercent *float64    `bson:"overallPercent,omitempty" json:"overallPercent,omitempty"`
	Timeline       interface{} `bson:"timeline,omitempty" json:"timeline,omitempty"`
	FileName       *string     `bson:"file_name,omitempty" json:"fileName,omitempty"`
	FileSize       *int64      `bson:"file_size,omitempty" json:"fileSize,omitempty"`
	Resolution     *string     `bson:"resolution,omitempty" json:"resolution,omitempty"`
	SourceType     *string     `bson:"sourceType,omitempty" json:"sourceType,omitempty"`
	M3U8URL        *string     `bson:"m3u8_url,omitempty" json:"m3u8Url,omitempty"`
	ProcessType    string      `bson:"processType" json:"processType" goose:"default:download"` // download
	Resolutions    []string    `bson:"resolutions,omitempty" json:"resolutions,omitempty"`
	Completed      []string    `bson:"completed,omitempty" json:"completed,omitempty"`
	Error          *string     `bson:"error,omitempty" json:"error,omitempty"`
	ErrorCategory  *string     `bson:"errorCategory,omitempty" json:"errorCategory,omitempty"`
	RetryCount     *int        `bson:"retryCount,omitempty" json:"retryCount,omitempty"`
	CreatedAt      time.Time   `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt      time.Time   `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// VideoProcessModel is the goose model for the "video_process" collection.
var VideoProcessModel = goose.NewModel[VideoProcess]("video_process")
