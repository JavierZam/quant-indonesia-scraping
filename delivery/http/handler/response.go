package handler

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta holds pagination metadata.
type Meta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}
