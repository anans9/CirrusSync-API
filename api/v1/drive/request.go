package drive

import "cirrussync-api/internal/models"

// DriveVolumeWrapper is a wrapper for models.DriveVolume that uses camelCase JSON tags
type DriveVolumeWrapper struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

// ToModel converts the wrapper to a models.DriveVolume
func (dvw *DriveVolumeWrapper) ToModel() *models.DriveVolume {
	return &models.DriveVolume{
		Name: dvw.Name,
		Hash: dvw.Hash,
	}
}

// DriveShareWrapper is a wrapper for models.DriveShare that uses camelCase JSON tags
type DriveShareWrapper struct {
	ShareKey                 string `json:"shareKey"`
	SharePassphrase          string `json:"sharePassphrase"`
	SharePassphraseSignature string `json:"sharePassphraseSignature"`
}

// ToModel converts the wrapper to a models.DriveShare
func (dsw *DriveShareWrapper) ToModel() *models.DriveShare {
	return &models.DriveShare{
		ShareKey:                 dsw.ShareKey,
		SharePassphrase:          dsw.SharePassphrase,
		SharePassphraseSignature: dsw.SharePassphraseSignature,
	}
}

// DriveShareMembershipWrapper is a wrapper for models.DriveShareMembership that uses camelCase JSON tags
type DriveShareMembershipWrapper struct {
	KeyPacket           string `json:"keyPacket"`
	KeyPacketSignature  string `json:"keyPacketSignature"`
	SessionKeySignature string `json:"sessionKeySignature"`
}

// ToModel converts the wrapper to a models.DriveShareMembership
func (dsmw *DriveShareMembershipWrapper) ToModel() *models.DriveShareMembership {
	return &models.DriveShareMembership{
		KeyPacket:           dsmw.KeyPacket,
		KeyPacketSignature:  dsmw.KeyPacketSignature,
		SessionKeySignature: dsmw.SessionKeySignature,
	}
}

// FilePropertiesWrapper is a wrapper for models.FileProperties that uses camelCase JSON tags
type FilePropertiesWrapper struct {
	ContentHash         string `json:"contentHash"`
	ContentKeyPacket    string `json:"contentKeyPacket"`
	ContentKeySignature string `json:"contentKeySignature"`
}

// ToModel converts the wrapper to a models.FileProperties
func (fpw *FilePropertiesWrapper) ToModel() *models.FileProperties {
	return &models.FileProperties{
		ContentHash:         fpw.ContentHash,
		ContentKeyPacket:    fpw.ContentKeyPacket,
		ContentKeySignature: fpw.ContentKeySignature,
	}
}

// FolderPropertiesWrapper is a wrapper for models.FolderProperties that uses camelCase JSON tags
type FolderPropertiesWrapper struct {
	NodeHashKey string `json:"nodeHashKey"`
}

// ToModel converts the wrapper to a models.FolderProperties
func (fpw *FolderPropertiesWrapper) ToModel() *models.FolderProperties {
	return &models.FolderProperties{
		NodeHashKey: fpw.NodeHashKey,
	}
}

// CreateDriveRequest uses wrapper structs instead of direct model references
type CreateDriveRequest struct {
	DriveVolume          DriveVolumeWrapper          `json:"driveVolume" binding:"required"`
	DriveShare           DriveShareWrapper           `json:"driveShare" binding:"required"`
	DriveShareMembership DriveShareMembershipWrapper `json:"driveShareMembership" binding:"required"`
}

// ToModelRequest converts the request to use actual model types
func (r *CreateDriveRequest) ToModelRequest() struct {
	DriveVolume          *models.DriveVolume
	DriveShare           *models.DriveShare
	DriveShareMembership *models.DriveShareMembership
} {
	return struct {
		DriveVolume          *models.DriveVolume
		DriveShare           *models.DriveShare
		DriveShareMembership *models.DriveShareMembership
	}{
		DriveVolume:          r.DriveVolume.ToModel(),
		DriveShare:           r.DriveShare.ToModel(),
		DriveShareMembership: r.DriveShareMembership.ToModel(),
	}
}

// CreateFolderRequest represents a request to create a new folder
type CreateFolderRequest struct {
	Name                    string  `json:"name" binding:"required"`
	Hash                    string  `json:"hash" binding:"required"`
	ParentId                *string `json:"parentId" binding:"required"`
	SignatureEmail          string  `json:"signatureEmail" binding:"required"`
	NodeKey                 string  `json:"nodeKey" binding:"required"`
	NodeHashKey             string  `json:"nodeHashKey" binding:"required"`
	NodePassphrase          string  `json:"nodePassphrase" binding:"required"`
	NodePassphraseSignature string  `json:"nodePassphraseSignature" binding:"required"`
}
