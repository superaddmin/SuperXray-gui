package entity

// APIErrorCode identifies stable API error classes for versioned endpoints.
type APIErrorCode string

const (
	APIErrorCodeUnauthorized APIErrorCode = "UNAUTHORIZED"
	APIErrorCodeForbidden    APIErrorCode = "FORBIDDEN"
	APIErrorCodeNotFound     APIErrorCode = "NOT_FOUND"
	APIErrorCodeRateLimited  APIErrorCode = "RATE_LIMITED"
	APIErrorCodeValidation   APIErrorCode = "VALIDATION_ERROR"
	APIErrorCodeInternal     APIErrorCode = "INTERNAL_ERROR"
)

// APIError is the error envelope used by versioned API endpoints.
type APIError struct {
	Code    APIErrorCode `json:"code"`
	Message string       `json:"message"`
	Details any          `json:"details,omitempty"`
}

// APIResponse is the unified response envelope for versioned API endpoints.
type APIResponse struct {
	Success   bool      `json:"success"`
	RequestID string    `json:"requestId,omitempty"`
	Data      any       `json:"data,omitempty"`
	Error     *APIError `json:"error,omitempty"`
}
