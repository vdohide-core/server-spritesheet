package models

import (
	"time"

	"github.com/zergolf1994/goose"
)

// WorkspaceMetadata holds metadata for a workspace.
type WorkspaceMetadata struct {
	BillingID *string    `bson:"billingId,omitempty" json:"billingId,omitempty" goose:"ref:user"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	DeletedBy *string    `bson:"deletedBy,omitempty" json:"deletedBy,omitempty" goose:"ref:user"`
}

// WorkspaceCapacity holds capacity info for a workspace.
type WorkspaceCapacity struct {
	Total      interface{} `bson:"total" json:"total"`
	Used       interface{} `bson:"used" json:"used"`
	Free       interface{} `bson:"free" json:"free"`
	Percentage float64     `bson:"percentage" json:"percentage"`
}

// WorkspacePlan holds plan info for a workspace.
type WorkspacePlan struct {
	PlanType     string      `bson:"planType" json:"planType"`
	StorageLimit interface{} `bson:"storageLimit,omitempty" json:"storageLimit,omitempty"`
	PriceTotal   *float64    `bson:"priceTotal,omitempty" json:"priceTotal,omitempty"`
	AdsEnabled   bool        `bson:"adsEnabled" json:"adsEnabled"`
	ExpiresAt    *time.Time  `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	DowngradeAt  *time.Time  `bson:"downgradeAt,omitempty" json:"downgradeAt,omitempty"`
}

// WorkspaceSettings holds settings for a workspace.
type WorkspaceSettings struct {
	RequestToJoin bool `bson:"requestToJoin" json:"requestToJoin"`
}

// Workspace represents a workspace record.
// Collection: "workspaces" | _id: String (UUID)
type Workspace struct {
	ID        string             `bson:"_id" json:"id" goose:"required,default:uuid"`
	CreatorID string             `bson:"creatorId" json:"creatorId" goose:"ref:user,index,required"`
	Status    string             `bson:"status" json:"status" goose:"default:pending"`
	Name      string             `bson:"name" json:"name" goose:"required"`
	Slug      string             `bson:"slug" json:"slug" goose:"unique,default:random(11)"`
	Image     *string            `bson:"image,omitempty" json:"image,omitempty"`
	Metadata  *WorkspaceMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Capacity  *WorkspaceCapacity `bson:"capacity,omitempty" json:"capacity,omitempty"`
	Plan      *WorkspacePlan     `bson:"plan,omitempty" json:"plan,omitempty"`
	Settings  *WorkspaceSettings `bson:"settings,omitempty" json:"settings,omitempty"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// WorkspaceModel is the goose model for the "workspaces" collection.
var WorkspaceModel = goose.NewModel[Workspace]("workspaces")

// WorkspaceMember represents a workspace membership.
// Collection: "workspace_members" | _id: String (UUID)
type WorkspaceMember struct {
	ID        string    `bson:"_id" json:"id" goose:"required,default:uuid"`
	SpaceID   string    `bson:"spaceId" json:"spaceId" goose:"ref:workspaces,index"`
	UserID    string    `bson:"userId" json:"userId" goose:"ref:user,index"`
	Role      string    `bson:"role" json:"role"`     // OWNER, ADMIN, MEMBER, VIEWER
	Status    string    `bson:"status" json:"status"` // PENDING, ACTIVE, INACTIVE, REJECTED
	InvitedBy *string   `bson:"invitedBy,omitempty" json:"invitedBy,omitempty" goose:"ref:user,index"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt" goose:"default:now"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt" goose:"default:now"`
}

// WorkspaceMemberModel is the goose model for the "workspace_members" collection.
var WorkspaceMemberModel = goose.NewModel[WorkspaceMember]("workspace_members")
