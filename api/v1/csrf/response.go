package csrf

import "cirrussync-api/pkg/status"

type CsrfResponse struct {
	Code      int16  `json:"code"`
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
}

type ErrorResponse struct {
	Code   int16  `json:"code"`
	Detail string `json:"detail"`
}

func NewResponse(token string, expiresAt int64) *CsrfResponse {
	return &CsrfResponse{
		Code:      status.StatusOK,
		Token:     token,
		ExpiresAt: expiresAt,
	}
}

func NewErrorResponse(code int16, message string) *ErrorResponse {
	return &ErrorResponse{
		Code:   code,
		Detail: message,
	}
}
