package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// PlayerConfig holds video player configuration for a custom domain.
type PlayerConfig struct {
	LogoImageURL    *string `bson:"logoImageUrl,omitempty" json:"logoImageUrl,omitempty"`
	LogoWebsiteURL  *string `bson:"logoWebsiteUrl,omitempty" json:"logoWebsiteUrl,omitempty"`
	LogoPosition    *string `bson:"logoPosition,omitempty" json:"logoPosition,omitempty"`
	PosterURL       *string `bson:"posterUrl,omitempty" json:"posterUrl,omitempty"`
	BaseColor       string  `bson:"baseColor" json:"baseColor"`
	DisplayTitle    bool    `bson:"displayTitle" json:"displayTitle"`
	AutoPlay        bool    `bson:"autoPlay" json:"autoPlay"`
	MuteSound       bool    `bson:"muteSound" json:"muteSound"`
	RepeatVideo     bool    `bson:"repeatVideo" json:"repeatVideo"`
	ContinuePlay    bool    `bson:"continuePlay" json:"continuePlay"`
	ContinuePlayArk bool    `bson:"continuePlayArk" json:"continuePlayArk"`
	Sharing         bool    `bson:"sharing" json:"sharing"`
	Captions        bool    `bson:"captions" json:"captions"`
	PlaybackRate    bool    `bson:"playbackRate" json:"playbackRate"`
	Keyboard        bool    `bson:"keyboard" json:"keyboard"`
	Download        bool    `bson:"download" json:"download"`
	PIP             bool    `bson:"pip" json:"pip"`
	ShowPreviewTime bool    `bson:"showPreviewTime" json:"showPreviewTime"`
	FastForward     bool    `bson:"fastForward" json:"fastForward"`
	Rewind          bool    `bson:"rewind" json:"rewind"`
	SeekStep        int     `bson:"seekStep" json:"seekStep"`
}

// DomainAds holds references to ads.
type DomainAds struct {
	Video  []string `bson:"video,omitempty" json:"video,omitempty" goose:"ref:ads"`
	Image  []string `bson:"image,omitempty" json:"image,omitempty" goose:"ref:ads"`
	Script []string `bson:"script,omitempty" json:"script,omitempty" goose:"ref:ads"`
}

// DomainDNS holds DNS configuration for domain verification.
type DomainDNS struct {
	RecordType        string     `bson:"recordType" json:"recordType"`
	Value             string     `bson:"value" json:"value"`
	TTL               int        `bson:"ttl" json:"ttl"`
	VerificationToken string     `bson:"verificationToken" json:"verificationToken"`
	LastVerified      *time.Time `bson:"lastVerified,omitempty" json:"lastVerified,omitempty"`
}

// CustomDomain represents a custom domain with player/ad config.
// Collection: "custom_domains" | _id: String (UUID)
type CustomDomain struct {
	ID        string        `bson:"_id" json:"id" goose:"required,default:uuid"`
	Enable    bool          `bson:"enable" json:"enable"`
	Name      string        `bson:"name" json:"name" goose:"required,unique"`
	Status    string        `bson:"status" json:"status" goose:"default:pending,index"`
	CreatorID *string       `bson:"creatorId,omitempty" json:"creatorId,omitempty" goose:"ref:user,index"`
	SpaceID   *string       `bson:"spaceId,omitempty" json:"spaceId,omitempty" goose:"ref:workspaces,index"`
	DNS       *DomainDNS    `bson:"dns,omitempty" json:"dns,omitempty"`
	Player    *PlayerConfig `bson:"player,omitempty" json:"player,omitempty"`
	Ads       *DomainAds    `bson:"ads,omitempty" json:"ads,omitempty"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt time.Time     `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// CustomDomainModel is the goose model for the "custom_domains" collection.
var CustomDomainModel = goose.NewModel[CustomDomain]("custom_domains")
