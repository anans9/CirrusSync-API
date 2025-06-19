package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"cirrussync-api/internal/utils"
)

// VolumeAllocation tracks how storage is allocated when sharing a volume
type VolumeAllocation struct {
	ID                   string  `gorm:"primaryKey;column:id"`
	VolumeID             string  `gorm:"column:volume_id;not null;index:idx_volume_allocations_volume_id"`
	UserID               string  `gorm:"column:user_id;not null;index:idx_volume_allocations_user_id"`
	AllocatedSize        int64   `gorm:"column:allocated_size;not null"` // Allocated size in bytes
	UsedSize             int64   `gorm:"column:used_size;default:0"`     // Used size in bytes
	AllocationPercentage float32 `gorm:"column:allocation_percentage"`   // Percentage of total volume
	IsOwner              bool    `gorm:"column:is_owner;default:false"`  // Whether this user is the volume owner
	CreatedAt            int64   `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt           int64   `gorm:"column:modified_at;autoCreateTime:false;not null"`
	Active               bool    `gorm:"column:active;default:true"` // Whether this allocation is active

	// Relationships
	Volume DriveVolume `gorm:"foreignKey:VolumeID"`
	User   User        `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for VolumeAllocation
func (VolumeAllocation) TableName() string {
	return "volume_allocations"
}

// BeforeCreate hook for VolumeAllocation
func (va *VolumeAllocation) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if va.ID == "" {
		va.ID = utils.GenerateLinkID()
	}
	if va.CreatedAt == 0 {
		va.CreatedAt = now
	}
	if va.ModifiedAt == 0 {
		va.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for VolumeAllocation
func (va *VolumeAllocation) BeforeUpdate(tx *gorm.DB) error {
	va.ModifiedAt = time.Now().Unix()
	return nil
}

// DriveVolume represents a storage volume in the drive system
type DriveVolume struct {
	ID        string `gorm:"primaryKey;column:id"`
	Name      string `gorm:"column:name;type:text;not null"`
	Hash      string `gorm:"column:hash;size:128"`
	State     int    `gorm:"column:state;default:1"` // 1=active, 0=inactive
	Size      int64  `gorm:"column:size;default:3221225472"`
	CreatedAt int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	UpdatedAt int64  `gorm:"column:updated_at;autoCreateTime:false;not null"`
	UserID    string `gorm:"column:user_id;index:idx_drive_volumes_user_id"` // Owner of the volume
	PlanType  string `gorm:"column:plan_type;size:50"`
	IsShared  bool   `gorm:"column:is_shared;default:false"`
	MaxUsers  int    `gorm:"column:max_users;default:5"` // Maximum number of users who can share this volume

	// Relationships
	User        User               `gorm:"foreignKey:UserID"`
	Shares      []DriveShare       `gorm:"foreignKey:VolumeID"`
	Allocations []VolumeAllocation `gorm:"foreignKey:VolumeID"`
}

// TableName specifies the table name for DriveVolume
func (DriveVolume) TableName() string {
	return "drive_volumes"
}

// BeforeCreate hook for DriveVolume
func (dv *DriveVolume) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if dv.ID == "" {
		dv.ID = utils.GenerateLinkID()
	}
	if dv.CreatedAt == 0 {
		dv.CreatedAt = now
	}
	if dv.UpdatedAt == 0 {
		dv.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook for DriveVolume
func (dv *DriveVolume) BeforeUpdate(tx *gorm.DB) error {
	dv.UpdatedAt = time.Now().Unix()
	return nil
}

// DriveShare represents a shared drive
type DriveShare struct {
	ID                       string `gorm:"primaryKey;column:id"`
	VolumeID                 string `gorm:"column:volume_id;not null;index:idx_drive_shares_volume_id"`
	UserID                   string `gorm:"column:user_id;not null;index:idx_drive_shares_user_id"`
	Type                     int    `gorm:"column:type;default:1"`
	State                    int    `gorm:"column:state;default:1"`
	Creator                  string `gorm:"column:creator;size:255;not null"`
	Locked                   bool   `gorm:"column:locked;default:false"`
	CreatedAt                int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt               int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`
	LinkID                   string `gorm:"column:link_id;not null;index:idx_drive_shares_link_id"`
	PermissionsMask          int    `gorm:"column:permissions_mask;default:0"`
	BlockSize                int64  `gorm:"column:block_size;default:4194304"`
	VolumeSoftDeleted        bool   `gorm:"column:volume_soft_deleted;default:false"`
	ShareKey                 string `gorm:"column:share_key;type:text;not null"`
	SharePassphrase          string `gorm:"column:share_passphrase;type:text;not null"`
	SharePassphraseSignature string `gorm:"column:share_passphrase_signature;type:text"`

	// Relationships
	User   User        `gorm:"foreignKey:UserID"`
	Volume DriveVolume `gorm:"foreignKey:VolumeID"`
	// AllocatedSize int64                  `gorm:"column:allocated_size"`
	Memberships []DriveShareMembership `gorm:"foreignKey:ShareID"`
	Items       []DriveItem            `gorm:"foreignKey:ShareID"`
}

// TableName specifies the table name for DriveShare
func (DriveShare) TableName() string {
	return "drive_shares"
}

// BeforeCreate hook for DriveShare
func (ds *DriveShare) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if ds.CreatedAt == 0 {
		ds.CreatedAt = now
	}
	if ds.ModifiedAt == 0 {
		ds.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for DriveShare
func (ds *DriveShare) BeforeUpdate(tx *gorm.DB) error {
	ds.ModifiedAt = time.Now().Unix()
	return nil
}

// DriveShareMembership represents a user's membership in a shared drive
type DriveShareMembership struct {
	ID                  string `gorm:"primaryKey;column:id"`
	ShareID             string `gorm:"column:share_id;not null;index:idx_share_members_share_id"`
	UserID              string `gorm:"column:user_id;not null;index:idx_share_members_user_id"`
	MemberID            string `gorm:"column:member_id;not null"`
	Inviter             string `gorm:"column:inviter;size:255"`
	Permissions         int    `gorm:"column:permissions;default:22"` // 22 = share+read+write
	KeyPacket           string `gorm:"column:key_packet;type:text;not null"`
	KeyPacketSignature  string `gorm:"column:key_packet_signature;type:text"`
	SessionKeySignature string `gorm:"column:session_key_signature;type:text"`
	State               int    `gorm:"column:state;default:1"`
	CreatedAt           int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt          int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`
	CanUnlock           *bool  `gorm:"column:can_unlock;default:null"`

	// Relationships
	Share DriveShare `gorm:"foreignKey:ShareID"`
	User  User       `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for DriveShareMembership
func (DriveShareMembership) TableName() string {
	return "drive_share_memberships"
}

// BeforeCreate hook for DriveShareMembership
func (dsm *DriveShareMembership) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if dsm.ID == "" {
		dsm.ID = utils.GenerateLinkID()
	}
	if dsm.CreatedAt == 0 {
		dsm.CreatedAt = now
	}
	if dsm.ModifiedAt == 0 {
		dsm.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for DriveShareMembership
func (dsm *DriveShareMembership) BeforeUpdate(tx *gorm.DB) error {
	dsm.ModifiedAt = time.Now().Unix()
	return nil
}

type FolderProperties struct {
	NodeHashKey          string `json:"node_hash_key"`
	NodeHashKeySignature string `json:"node_hash_key_signature"`
}

type FileProperties struct {
	ContentHash         string `json:"content_hash"`
	ContentKeyPacket    string `json:"content_key_packet"`
	ContentKeySignature string `json:"content_key_signature"`
}

// DriveItem represents a file or folder in a drive
type DriveItem struct {
	ID                      string            `gorm:"primaryKey;column:id"`
	ParentID                *string           `gorm:"column:parent_id;index:idx_drive_items_parent_id"`
	ShareID                 string            `gorm:"column:share_id;not null;index:idx_drive_items_share_id"`
	VolumeID                string            `gorm:"column:volume_id;not null"`
	Type                    int               `gorm:"column:type;not null;index:idx_drive_items_type"` // 1=folder, 2=file
	Name                    string            `gorm:"column:name;type:text;not null"`
	Hash                    string            `gorm:"column:hash;size:128"`
	NameSignatureEmail      string            `gorm:"column:name_signature_email;size:255"`
	State                   int               `gorm:"column:state;default:1"`
	Size                    int64             `gorm:"column:size;default:0"`
	MimeType                *string           `gorm:"column:mime_type;size:100;default:null"`
	NodeKey                 string            `gorm:"column:node_key;type:text;not null"`
	NodePassphrase          string            `gorm:"column:node_passphrase;type:text;not null"`
	NodePassphraseSignature string            `gorm:"column:node_passphrase_signature;type:text"`
	SignatureEmail          string            `gorm:"column:signature_email;size:255"`
	CreatedAt               int64             `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt              int64             `gorm:"column:modified_at;autoCreateTime:false;not null"`
	IsTrashed               bool              `gorm:"column:is_trashed;default:false"`
	TrashedAt               *int64            `gorm:"column:trashed_at;default:null"`
	Permissions             int               `gorm:"column:permissions;default:7"`
	PermissionExpiresAt     *int64            `gorm:"column:permission_expires_at;default:null"`
	IsShared                bool              `gorm:"column:is_shared;default:false"`
	SharingDetails          *json.RawMessage  `gorm:"column:sharing_details;type:jsonb"`
	ShareURLs               *json.RawMessage  `gorm:"column:share_urls;type:jsonb"`
	ShareIDs                *json.RawMessage  `gorm:"column:share_ids;type:jsonb"`
	NbURLs                  int               `gorm:"column:nb_urls;default:0"`
	URLsExpired             int               `gorm:"column:urls_expired;default:0"`
	FileProperties          *FileProperties   `gorm:"column:file_properties;type:jsonb;serializer:json;default:'{}'"`
	FolderProperties        *FolderProperties `gorm:"column:folder_properties;type:jsonb;serializer:json;default:'{}'"`
	Xattrs                  *string           `gorm:"column:xattrs;type:text;default:null"`

	// Relationships
	Parent    *DriveItem     `gorm:"foreignKey:ParentID"`
	Children  []DriveItem    `gorm:"foreignKey:ParentID"`
	Share     DriveShare     `gorm:"foreignKey:ShareID"`
	Revisions []FileRevision `gorm:"foreignKey:ItemID"`
}

// TableName specifies the table name for DriveItem
func (DriveItem) TableName() string {
	return "drive_items"
}

// BeforeCreate hook for DriveItem
func (di *DriveItem) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if di.ID == "" {
		di.ID = utils.GenerateLinkID()
	}
	if di.CreatedAt == 0 {
		di.CreatedAt = now
	}
	if di.ModifiedAt == 0 {
		di.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for DriveItem
func (di *DriveItem) BeforeUpdate(tx *gorm.DB) error {
	di.ModifiedAt = time.Now().Unix()
	return nil
}
