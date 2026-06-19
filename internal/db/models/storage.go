package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// StorageLocalSSH holds SSH config for local storage.
type StorageLocalSSH struct {
	Username string `bson:"username,omitempty" json:"username,omitempty"`
	Password string `bson:"password,omitempty" json:"-"`
	Port     int    `bson:"port" json:"port"`
}

// StorageLocalConfig holds config for Nginx VOD local storage.
type StorageLocalConfig struct {
	Host string           `bson:"host" json:"host"`
	Port int              `bson:"port" json:"port"`
	Path string           `bson:"path" json:"path"`
	SSH  *StorageLocalSSH `bson:"ssh,omitempty" json:"ssh,omitempty"`
}

// StorageS3Config holds config for S3-compatible storage.
type StorageS3Config struct {
	Endpoint        *string `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
	Region          string  `bson:"region" json:"region"`
	Bucket          string  `bson:"bucket" json:"bucket"`
	Prefix          string  `bson:"prefix" json:"prefix"`
	AccessKeyID     string  `bson:"accessKeyId" json:"-"`
	SecretAccessKey string  `bson:"secretAccessKey" json:"-"`
	ForcePathStyle  bool    `bson:"forcePathStyle" json:"forcePathStyle"`
}

// StorageCapacity holds storage capacity stats.
type StorageCapacity struct {
	Total      interface{} `bson:"total" json:"total"`
	Used       interface{} `bson:"used" json:"used"`
	Free       interface{} `bson:"free" json:"free"`
	Percentage float64     `bson:"percentage" json:"percentage"`
}

// Storage represents a storage backend (local or S3).
// Collection: "storages" | _id: String (UUID)
type Storage struct {
	ID          string              `bson:"_id" json:"id" goose:"required,default:uuid"`
	Name        string              `bson:"name" json:"name" goose:"required"`
	Enable      bool                `bson:"enable" json:"enable"`
	Type        string              `bson:"type" json:"type"`                             // local, s3
	Status      string              `bson:"status" json:"status" goose:"default:offline"` // online, offline, error, maintenance
	Local       *StorageLocalConfig `bson:"local,omitempty" json:"local,omitempty"`
	S3          *StorageS3Config    `bson:"s3,omitempty" json:"s3,omitempty"`
	PublicURL   *string             `bson:"publicUrl,omitempty" json:"publicUrl,omitempty"`
	Accepts     []string            `bson:"accepts" json:"accepts"` // upload, video, image, other
	HeartbeatAt *time.Time          `bson:"heartbeatAt,omitempty" json:"heartbeatAt,omitempty"`
	Capacity    *StorageCapacity    `bson:"capacity,omitempty" json:"capacity,omitempty"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt   time.Time           `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// StorageModel is the goose model for the "storages" collection.
var StorageModel = goose.NewModel[Storage]("storages")

// GetPath returns the storage base path.
func (s *Storage) GetPath() string {
	if s.Local != nil && s.Local.Path != "" {
		return s.Local.Path
	}
	return "/home/files"
}

// GetHost returns the storage host.
func (s *Storage) GetHost() string {
	if s.Local != nil {
		return s.Local.Host
	}
	return ""
}

// GetPort returns the internal HTTP port for local storage.
func (s *Storage) GetPort() int {
	if s.Local != nil {
		return s.Local.Port
	}
	return 0
}

// HasSSHCredentials checks if storage has valid SSH credentials.
func (s *Storage) HasSSHCredentials() bool {
	if s.Local == nil || s.Local.SSH == nil {
		return false
	}
	return s.Local.SSH.Username != "" && s.Local.SSH.Password != "" && s.Local.SSH.Port > 0
}

// IsOnline checks if storage is enabled and online.
func (s *Storage) IsOnline() bool {
	return s.Enable && s.Status == StorageStatusOnline
}
