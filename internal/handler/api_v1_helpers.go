package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tumble/internal/data"
)

// writeJSON writes a JSON response with the given status code and data.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeAPIError writes a structured error response using APIErrorResponse.
func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	resp := APIErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
		},
	}
	writeJSON(w, status, resp)
}

// ValidationErrorResponse is used for validation errors with field details.
type ValidationErrorResponse struct {
	Error ValidationError `json:"error"`
}

// ValidationError contains the validation error details.
type ValidationError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

// writeValidationError writes a 422 Unprocessable Entity response with validation details.
func writeValidationError(w http.ResponseWriter, details map[string]string) {
	resp := ValidationErrorResponse{
		Error: ValidationError{
			Code:    "validation_error",
			Message: "Validation failed",
			Details: details,
		},
	}
	writeJSON(w, http.StatusUnprocessableEntity, resp)
}

// parseIntParam parses an integer query parameter with default and max value constraints.
// Returns defaultVal if the parameter is missing, invalid, or negative.
// Returns maxVal if the parameter exceeds maxVal.
func parseIntParam(r *http.Request, name string, defaultVal, maxVal int) int {
	paramStr := r.URL.Query().Get(name)
	if paramStr == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(paramStr)
	if err != nil {
		return defaultVal
	}

	if val < 0 {
		return defaultVal
	}

	if val > maxVal {
		return maxVal
	}

	return val
}

// wantsJSON returns true if the request indicates a preference for JSON response.
// It checks for .json suffix in the path or application/json in the Accept header.
func wantsJSON(r *http.Request) bool {
	path := r.URL.Path

	// Check for .json suffix (before any query string)
	if strings.HasSuffix(path, ".json") {
		return true
	}

	// Check Accept header
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	// Check if Accept contains application/json
	return strings.Contains(accept, "application/json")
}

// wantsPlainText returns true if the request indicates a preference for plain text response.
// It checks for .txt suffix in the path or text/plain in the Accept header.
func wantsPlainText(r *http.Request) bool {
	path := r.URL.Path

	// Check for .txt suffix
	if strings.HasSuffix(path, ".txt") {
		return true
	}

	// Check Accept header
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	// Check if Accept contains text/plain
	return strings.Contains(accept, "text/plain")
}

// trimFormatSuffix removes .json or .txt suffix from a path.
func trimFormatSuffix(path string) string {
	if strings.HasSuffix(path, ".json") {
		return strings.TrimSuffix(path, ".json")
	}
	if strings.HasSuffix(path, ".txt") {
		return strings.TrimSuffix(path, ".txt")
	}
	return path
}

// parseClientFilter extracts client filter parameters from the request query string.
func parseClientFilter(r *http.Request) (data.ClientFilter, error) {
	var f data.ClientFilter
	if st := r.URL.Query().Get("client_type"); st != "" {
		f.ClientType = &st
	}
	if sn := r.URL.Query().Get("client_network"); sn != "" {
		f.ClientNetwork = &sn
	}
	if sc := r.URL.Query().Get("client_channel"); sc != "" {
		f.ClientChannel = &sc
	}
	if err := f.Validate(); err != nil {
		return f, err
	}
	return f, nil
}

// isAuthorizedAPIKey checks if the request has a valid API key.
// Uses X-API-Key header only (no query param auth - per design doc).
// Always allows localhost requests.
func isAuthorizedAPIKey(r *http.Request, secret string) bool {
	// Localhost is always allowed
	remoteAddr := r.RemoteAddr
	if strings.HasPrefix(remoteAddr, "127.0.0.1") ||
		strings.HasPrefix(remoteAddr, "localhost") ||
		strings.HasPrefix(remoteAddr, "[::1]") {
		return true
	}

	// No secret configured
	if secret == "" {
		log.Printf("API auth failed: no secret configured")
		return false
	}

	// Check X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		log.Printf("API auth failed: missing X-API-Key header")
		return false
	}

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(apiKey), []byte(secret)) == 1
}
