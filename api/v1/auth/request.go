package auth

import "cirrussync-api/internal/user"

// LoginInitRequest represents the request to initialize SRP authentication
type LoginInitRequest struct {
	Email        string `json:"email" binding:"required,email"`
	ClientPublic string `json:"clientPublic" binding:"required"`
}

// LoginVerifyRequest represents the request to verify SRP authentication
type LoginVerifyRequest struct {
	SessionID   string `json:"sessionId" binding:"required"`
	ClientProof string `json:"clientProof" binding:"required"`
}

// SignupRequest represents the request for user registration
type SignupRequest struct {
	Email       string       `json:"email" binding:"required,email"`
	Username    string       `json:"username" binding:"required,min=3,max=30"`
	SRPSalt     string       `json:"srpSalt" binding:"required"`
	SRPVerifier string       `json:"srpVerifier" binding:"required"`
	Keys        user.UserKey `json:"keys" binding:"required"`
}

// ChangePasswordRequest represents the request body for changing a password
type ChangePasswordRequest struct {
	OldSRPSalt     string `json:"oldSrpSalt" binding:"required"`
	OldSRPVerifier string `json:"oldSrpVerifier" binding:"required"`
	NewSRPSalt     string `json:"newSrpSalt" binding:"required"`
	NewSRPVerifier string `json:"newSrpVerifier" binding:"required"`
}
