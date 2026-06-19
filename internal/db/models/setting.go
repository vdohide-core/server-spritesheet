package models

import (
	"fmt"
	"time"

	"github.com/zergolf1994/goose"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Setting keys
const (
	SettingDownloadEnabled  = "download_enabled"
	SettingURLScraping      = "url_scraping"
	SettingTranscodeEnabled = "transcode_enabled"
	SettingGpuEnabled       = "transcode_gpu_enabled"
	SettingTranscodeSort    = "transcode_sort"
	SettingDomainContent    = "domain_content"
	SettingCfZoneID         = "cf_zone_id"
	SettingCfApiToken       = "cf_api_token"

	// Spritesheet settings
	SettingSpritesheetEnabled = "spritesheet_enabled"
)

// Setting represents a system setting key-value pair.
// Collection: "settings" | _id: String (UUID)
type Setting struct {
	ID        string      `bson:"_id" json:"id" goose:"required,default:uuid"`
	Name      string      `bson:"name" json:"name" goose:"required,unique"`
	Value     interface{} `bson:"value" json:"value"`
	CreatedAt time.Time   `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt time.Time   `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// SettingModel is the goose model for the "settings" collection.
var SettingModel = goose.NewModel[Setting]("settings")

// GetBool returns Value as bool, defaults to defaultVal if not bool.
func (s *Setting) GetBool(defaultVal bool) bool {
	if v, ok := s.Value.(bool); ok {
		return v
	}
	if v, ok := s.Value.(string); ok {
		return v == "true" || v == "1"
	}
	switch v := s.Value.(type) {
	case int:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	}
	return defaultVal
}

// GetString returns Value as string.
func (s *Setting) GetString(defaultVal string) string {
	if v, ok := s.Value.(string); ok {
		return v
	}
	return defaultVal
}

// GetInt returns Value as int.
func (s *Setting) GetInt(defaultVal int) int {
	switch v := s.Value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultVal
}

// GetStringSlice returns Value as []string.
func (s *Setting) GetStringSlice() []string {
	var arr []interface{}
	switch v := s.Value.(type) {
	case primitive.A:
		arr = []interface{}(v)
	case []interface{}:
		arr = v
	default:
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		switch str := v.(type) {
		case string:
			result = append(result, str)
		default:
			result = append(result, fmt.Sprintf("%v", v))
		}
	}
	return result
}
