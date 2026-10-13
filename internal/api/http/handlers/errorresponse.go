package handlers

// ErrorResponse is the JSON body every failed request is answered with; nil Details encodes as null.
type ErrorResponse struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}
