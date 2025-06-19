package drive

import (
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/redis"
)

// Service handles drive operations
type Service struct {
	repo        Repository
	redisClient *redis.Client
	logger      *logger.Logger
}

// ShareWithMemberships represents a share with its memberships
type ShareWithMemberships struct {
	Share       *models.DriveShare
	Memberships []*models.DriveShareMembership
}

type ShareKeys struct {
	ShareKey                 string
	SharePassphrase          string
	SharePassphraseSignature string
}

type DriveItemKeys struct {
	NodeKey                 string
	NodePassphrase          string
	NodePassphraseSignature string
}

type DriveShareMemberKeys struct {
	KeyPacket           string
	KeyPacketSignature  string
	SessionKeySignature string
}

type EncryptionKeys struct {
	Share            ShareKeys
	Drive            DriveItemKeys
	DriveShareMember DriveShareMemberKeys
}
