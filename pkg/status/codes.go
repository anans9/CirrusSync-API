package status

// Status codes for API responses
// 1000-1999: Success codes
// 2000-2999: Challenge/Verification codes
// 4000-4999: Client error codes
// 5000-5999: Server error codes

const (
	// Success codes (1000-1999)
	StatusOK                 int16 = 1000
	StatusCreated            int16 = 1001
	StatusAccepted           int16 = 1002
	StatusUpdated            int16 = 1003
	StatusDeleted            int16 = 1004
	StatusLoginSuccess       int16 = 1010
	StatusSignupSuccess      int16 = 1011
	StatusTokenRefreshed     int16 = 1012
	StatusLogoutSuccess      int16 = 1013
	StatusPasswordChanged    int16 = 1014
	StatusEmailVerified      int16 = 1015
	StatusPaymentSuccess     int16 = 1020
	StatusSubscriptionActive int16 = 1021
	StatusFileUploaded       int16 = 1030
	StatusFileDownloaded     int16 = 1031
	StatusFileSynced         int16 = 1032
	StatusShareCreated       int16 = 1040
	StatusDeviceAuthorized   int16 = 1050

	// Challenge codes (2000-2999)
	StatusChallengeIssued        int16 = 2000
	StatusMFARequired            int16 = 2001
	StatusEmailVerificationSent  int16 = 2002
	StatusPasswordResetIssued    int16 = 2003
	StatusDeviceVerificationSent int16 = 2004
	StatusSRPChallengeIssued     int16 = 2010
	StatusAdditionalAuthRequired int16 = 2020
	StatusPaymentRequired        int16 = 2030

	// Client error codes (4000-4999)
	StatusBadRequest           int16 = 4000
	StatusUnauthorized         int16 = 4001
	StatusForbidden            int16 = 4002
	StatusNotFound             int16 = 4003
	StatusConflict             int16 = 4004
	StatusTooManyRequests      int16 = 4005
	StatusValidationFailed     int16 = 4010
	StatusInvalidCredentials   int16 = 4011
	StatusInvalidToken         int16 = 4012
	StatusTokenExpired         int16 = 4013
	StatusInvalidEmail         int16 = 4020
	StatusEmailAlreadyExists   int16 = 4021
	StatusWeakPassword         int16 = 4022
	StatusAccountLocked        int16 = 4023
	StatusMFAFailed            int16 = 4030
	StatusInvalidMFACode       int16 = 4031
	StatusCSRFTokenMismatch    int16 = 4040
	StatusSubscriptionExpired  int16 = 4050
	StatusPaymentDeclined      int16 = 4051
	StatusStorageQuotaExceeded int16 = 4060
	StatusFileAlreadyExists    int16 = 4061
	StatusInvalidDeviceID      int16 = 4070
	StatusSessionExpired       int16 = 4051
	StatusInvalidSession       int16 = 4052

	// Server error codes (5000-5999)
	StatusInternalServerError  int16 = 5000
	StatusNotImplemented       int16 = 5001
	StatusServiceUnavailable   int16 = 5002
	StatusDBError              int16 = 5010
	StatusRedisError           int16 = 5011
	StatusEncryptionError      int16 = 5020
	StatusJWTError             int16 = 5030
	StatusSRPError             int16 = 5031
	StatusFileSystemError      int16 = 5040
	StatusExternalServiceError int16 = 5050
	StatusPaymentGatewayError  int16 = 5051
	StatusMailServiceError     int16 = 5052
)

// Code returns a descriptive string for the status code
func CodeToString(code int16) string {
	switch code {
	// Success codes
	case StatusOK:
		return "OK"
	case StatusCreated:
		return "Resource created successfully"
	case StatusAccepted:
		return "Request accepted for processing"
	case StatusUpdated:
		return "Resource updated successfully"
	case StatusDeleted:
		return "Resource deleted successfully"
	case StatusLoginSuccess:
		return "Login successful"
	case StatusSignupSuccess:
		return "Signup successful"
	case StatusTokenRefreshed:
		return "Token refreshed successfully"
	case StatusLogoutSuccess:
		return "Logout successful"

	// Challenge codes
	case StatusChallengeIssued:
		return "Challenge issued"
	case StatusMFARequired:
		return "Multi-factor authentication required"
	case StatusSRPChallengeIssued:
		return "SRP authentication challenge issued"

	// Client error codes
	case StatusBadRequest:
		return "Bad request"
	case StatusUnauthorized:
		return "Unauthorized"
	case StatusForbidden:
		return "Forbidden"
	case StatusNotFound:
		return "Resource not found"
	case StatusConflict:
		return "Resource conflict"
	case StatusInvalidCredentials:
		return "Invalid credentials"
	case StatusTokenExpired:
		return "Token has expired"

	// Server error codes
	case StatusInternalServerError:
		return "Internal server error"
	case StatusNotImplemented:
		return "Not implemented"
	case StatusServiceUnavailable:
		return "Service unavailable"
	case StatusDBError:
		return "Database error"
	case StatusSRPError:
		return "SRP authentication error"

	default:
		return "Unknown status code"
	}
}

// IsSuccess returns true if the code is a success code
func IsSuccess(code int16) bool {
	return code >= 1000 && code < 2000
}

// IsChallenge returns true if the code is a challenge code
func IsChallenge(code int16) bool {
	return code >= 2000 && code < 3000
}

// IsClientError returns true if the code is a client error code
func IsClientError(code int16) bool {
	return code >= 4000 && code < 5000
}

// IsServerError returns true if the code is a server error code
func IsServerError(code int16) bool {
	return code >= 5000 && code < 6000
}
