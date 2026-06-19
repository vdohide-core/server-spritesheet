package models

// ─── User Roles ──────────────────────────────────────────────────────

const (
	UserRoleUser       = "user"
	UserRoleAdmin      = "admin"
	UserRoleSuperAdmin = "super_admin"
	UserRoleDeveloper  = "developer"
)

// ─── Workspace Member Roles ──────────────────────────────────────────

const (
	WorkspaceMemberRoleOwner  = "owner"
	WorkspaceMemberRoleAdmin  = "admin"
	WorkspaceMemberRoleMember = "member"
	WorkspaceMemberRoleViewer = "viewer"
)

// ─── File Types ──────────────────────────────────────────────────────

const (
	FileTypeFolder = "folder"
	FileTypeVideo  = "video"
	FileTypeImage  = "image"
	FileTypeOther  = "other"
	FileTypeSpace  = "space"
)

// ─── File Statuses ───────────────────────────────────────────────────

const (
	FileStatusWaiting    = "waiting"
	FileStatusProcessing = "processing"
	FileStatusReady      = "ready"
	FileStatusError      = "error"
)

// ─── File Source Types ───────────────────────────────────────────────

const (
	FileSourceTypeUpload  = "upload"
	FileSourceTypeYoutube = "youtube"
	FileSourceTypeVimeo   = "vimeo"
	FileSourceTypeOther   = "other"
)

// ─── Media Types ─────────────────────────────────────────────────────

const (
	MediaTypeVideo     = "video"
	MediaTypeAudio     = "audio"
	MediaTypeSubtitle  = "subtitle"
	MediaTypeThumbnail = "thumbnail"
	MediaTypeImage     = "image"
	MediaTypeDocument  = "document"
	MediaTypeOther     = "other"
)

// ─── Ingest Source Types ─────────────────────────────────────────────

const (
	IngestSourceTypeUpload    = "upload"
	IngestSourceTypeRemote    = "remote"
	IngestSourceTypeGDrive    = "gdrive"
	IngestSourceTypeS3Import  = "s3_import"
	IngestSourceTypeProcessed = "processed"
)

// ─── Storage Types ───────────────────────────────────────────────────

const (
	StorageTypeLocal = "local"
	StorageTypeS3    = "s3"
)

// ─── Storage Statuses ────────────────────────────────────────────────

const (
	StorageStatusOnline      = "online"
	StorageStatusOffline     = "offline"
	StorageStatusError       = "error"
	StorageStatusMaintenance = "maintenance"
)

// ─── Storage Accepts ─────────────────────────────────────────────────

const (
	StorageAcceptUpload = "upload"
	StorageAcceptVideo  = "video"
	StorageAcceptImage  = "image"
	StorageAcceptOther  = "other"
)

// ─── Resolution ──────────────────────────────────────────────────────

const (
	ResolutionOriginal = "original"
	Resolution1080     = "1080"
	Resolution720      = "720"
	Resolution480      = "480"
	Resolution360      = "360"
)

// ─── File Name Patterns ──────────────────────────────────────────────

const (
	FileNameOriginal = "file_original.mp4"
	FileName1080     = "file_1080.mp4"
	FileName720      = "file_720.mp4"
	FileName480      = "file_480.mp4"
	FileName360      = "file_360.mp4"
)

// ResolutionToFileName maps resolution string to file name.
var ResolutionToFileName = map[string]string{
	Resolution1080: FileName1080,
	Resolution720:  FileName720,
	Resolution480:  FileName480,
	Resolution360:  FileName360,
}

// ResolutionToShortSide maps resolution string to short side pixel count.
var ResolutionToShortSide = map[string]int{
	Resolution1080: 1080,
	Resolution720:  720,
	Resolution480:  480,
	Resolution360:  360,
}

// ─── Sprite / Thumbnail ──────────────────────────────────────────────

const (
	SpriteVTTName = "sprite.vtt"
	SpriteZipName = "sprite.zip"
)

// ─── Process Types ───────────────────────────────────────────────────

const (
	ProcessTypeDownload    = "download"
	ProcessTypeTranscode   = "transcode"
	ProcessTypeSubtitle    = "subtitle"
	ProcessTypeSpritesheet = "spritesheet"
)

// ─── Domain Statuses ─────────────────────────────────────────────────

const (
	DomainStatusPending = "pending"
	DomainStatusActive  = "active"
	DomainStatusFailed  = "failed"
	DomainStatusExpired = "expired"
)

// ─── DMCA Statuses ───────────────────────────────────────────────────

const (
	DmcaStatusPending       = "pending"
	DmcaStatusReviewing     = "reviewing"
	DmcaStatusApproved      = "approved"
	DmcaStatusRejected      = "rejected"
	DmcaStatusCounterNotice = "counter_notice"
)

// ─── DMCA Types ──────────────────────────────────────────────────────

const (
	DmcaTypeCopyright = "copyright"
	DmcaTypeTrademark = "trademark"
	DmcaTypeOther     = "other"
)
