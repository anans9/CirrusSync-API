package models

import (
	utils "cirrussync-api/internal/utils"
	"time"

	"gorm.io/gorm"
)

// FileRevision represents a revision of a file
type FileRevision struct {
	ID             string `gorm:"primaryKey;column:id"`
	ItemID         string `gorm:"column:item_id;not null;index:idx_file_revisions_item_id"`
	Size           int64  `gorm:"column:size"`
	CreatedAt      int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	State          int    `gorm:"column:state;default:1"`
	SignatureEmail string `gorm:"column:signature_email;size:255"`

	// Relationships
	Item       DriveItem        `gorm:"foreignKey:ItemID"`
	Thumbnails []DriveThumbnail `gorm:"foreignKey:RevisionID"`
	Blocks     []FileBlock      `gorm:"foreignKey:RevisionID"`
}

// TableName specifies the table name for FileRevision
func (FileRevision) TableName() string {
	return "file_revisions"
}

// BeforeCreate hook for FileRevision
func (fr *FileRevision) BeforeCreate(tx *gorm.DB) error {
	if fr.ID == "" {
		fr.ID = utils.GenerateLinkID()
	}
	if fr.CreatedAt == 0 {
		fr.CreatedAt = time.Now().Unix()
	}
	return nil
}

// DriveThumbnail represents a thumbnail for a file
type DriveThumbnail struct {
	ID                 string `gorm:"primaryKey;column:id"`
	RevisionID         string `gorm:"column:revision_id;not null;index:idx_drive_thumbnails_revision_id"`
	Type               int    `gorm:"column:type;default:1"`
	Hash               string `gorm:"column:hash;size:128"`
	Size               int64  `gorm:"column:size"`
	StoragePath        string `gorm:"column:storage_path;size:1024"`
	StorageBucket      string `gorm:"column:storage_bucket;size:255"`
	StorageRegion      string `gorm:"column:storage_region;size:50"`
	ThumbnailSignature string `gorm:"column:thumbnail_signature;type:text"`
	CreatedAt          int64  `gorm:"column:created_at;autoCreateTime:false;not null"`

	// Relationships
	Revision FileRevision `gorm:"foreignKey:RevisionID"`
}

// TableName specifies the table name for DriveThumbnail
func (DriveThumbnail) TableName() string {
	return "drive_thumbnails"
}

// FileBlock represents a block of a file
type FileBlock struct {
	ID                 string `gorm:"primaryKey;column:id"`
	RevisionID         string `gorm:"column:revision_id;not null;index:idx_file_blocks_revision_id"`
	Index              int    `gorm:"column:index;default:0"`
	Size               int64  `gorm:"column:size"`
	Hash               string `gorm:"column:hash;size:128;index:idx_file_blocks_hash"`
	StoragePath        string `gorm:"column:storage_path;size:1024"`
	StorageBucket      string `gorm:"column:storage_bucket;size:255"`
	StorageRegion      string `gorm:"column:storage_region;size:50"`
	KeyPacket          string `gorm:"column:key_packet;type:text"`
	KeyPacketSignature string `gorm:"column:key_packet_signature;type:text"`
	UploadComplete     bool   `gorm:"column:upload_complete;default:false"`
	UploadTime         int64  `gorm:"column:upload_time"`
	CreatedAt          int64  `gorm:"column:created_at;autoCreateTime:false;not null"`

	// Relationships
	Revision FileRevision `gorm:"foreignKey:RevisionID"`
}

// TableName specifies the table name for FileBlock
func (FileBlock) TableName() string {
	return "file_blocks"
}

// BeforeCreate hook for FileBlock
func (fb *FileBlock) BeforeCreate(tx *gorm.DB) error {
	if fb.ID == "" {
		fb.ID = utils.GenerateLinkID()
	}
	if fb.CreatedAt == 0 {
		fb.CreatedAt = time.Now().Unix()
	}
	return nil
}
