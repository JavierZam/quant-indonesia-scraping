package handler

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError represents a structured error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta holds pagination metadata.
type Meta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

// Error code constants.
const (
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeInternal     = "INTERNAL_ERROR"
	ErrCodeBadRequest   = "BAD_REQUEST"
	ErrCodeRateLimited  = "RATE_LIMITED"
	ErrCodeConflict     = "CONFLICT"
)

// NewErrorResponse creates an error APIResponse.
func NewErrorResponse(code, message string) APIResponse {
	return APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	}
}

// NewSuccessResponse creates a success APIResponse with data.
func NewSuccessResponse(data interface{}, meta *Meta) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}
