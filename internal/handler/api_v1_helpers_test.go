package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "simple object",
			status:         http.StatusOK,
			data:           map[string]string{"message": "hello"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"hello"}`,
		},
		{
			name:           "created status",
			status:         http.StatusCreated,
			data:           map[string]int{"id": 42},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":42}`,
		},
		{
			name:           "empty object",
			status:         http.StatusOK,
			data:           map[string]string{},
			expectedStatus: http.StatusOK,
			expectedBody:   `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.status, tt.data)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			body := w.Body.String()
			// Trim newline added by json.Encoder
			if len(body) > 0 && body[len(body)-1] == '\n' {
				body = body[:len(body)-1]
			}
			if body != tt.expectedBody {
				t.Errorf("expected body %s, got %s", tt.expectedBody, body)
			}
		})
	}
}

func TestWriteAPIError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		code           string
		message        string
		expectedStatus int
	}{
		{
			name:           "not found error",
			status:         http.StatusNotFound,
			code:           "not_found",
			message:        "Resource not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "bad request error",
			status:         http.StatusBadRequest,
			code:           "invalid_request",
			message:        "Invalid request body",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "internal server error",
			status:         http.StatusInternalServerError,
			code:           "internal_error",
			message:        "An internal error occurred",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeAPIError(w, tt.status, tt.code, tt.message)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			var resp APIErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if resp.Error.Code != tt.code {
				t.Errorf("expected error code %s, got %s", tt.code, resp.Error.Code)
			}
			if resp.Error.Message != tt.message {
				t.Errorf("expected error message %s, got %s", tt.message, resp.Error.Message)
			}
		})
	}
}

func TestWriteValidationError(t *testing.T) {
	tests := []struct {
		name    string
		details map[string]string
	}{
		{
			name: "single field error",
			details: map[string]string{
				"url": "URL is required",
			},
		},
		{
			name: "multiple field errors",
			details: map[string]string{
				"url":   "URL is required",
				"title": "Title cannot be empty",
			},
		},
		{
			name:    "empty details",
			details: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeValidationError(w, tt.details)

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			errorObj, ok := resp["error"].(map[string]interface{})
			if !ok {
				t.Fatal("expected error object in response")
			}

			if errorObj["code"] != "validation_error" {
				t.Errorf("expected error code validation_error, got %v", errorObj["code"])
			}

			details, ok := errorObj["details"].(map[string]interface{})
			if !ok {
				t.Fatal("expected details object in error")
			}

			for field, msg := range tt.details {
				if details[field] != msg {
					t.Errorf("expected field %s to have message %s, got %v", field, msg, details[field])
				}
			}
		})
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name       string
		paramName  string
		paramValue string
		defaultVal int
		maxVal     int
		expected   int
	}{
		{
			name:       "use default when missing",
			paramName:  "limit",
			paramValue: "",
			defaultVal: 20,
			maxVal:     100,
			expected:   20,
		},
		{
			name:       "valid value",
			paramName:  "limit",
			paramValue: "50",
			defaultVal: 20,
			maxVal:     100,
			expected:   50,
		},
		{
			name:       "exceeds max",
			paramName:  "limit",
			paramValue: "200",
			defaultVal: 20,
			maxVal:     100,
			expected:   100,
		},
		{
			name:       "negative value uses default",
			paramName:  "limit",
			paramValue: "-5",
			defaultVal: 20,
			maxVal:     100,
			expected:   20,
		},
		{
			name:       "invalid value uses default",
			paramName:  "limit",
			paramValue: "abc",
			defaultVal: 20,
			maxVal:     100,
			expected:   20,
		},
		{
			name:       "zero value is valid",
			paramName:  "offset",
			paramValue: "0",
			defaultVal: 10,
			maxVal:     100,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.paramName+"="+tt.paramValue, nil)
			result := parseIntParam(req, tt.paramName, tt.defaultVal, tt.maxVal)

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestWantsJSON(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		accept   string
		expected bool
	}{
		{
			name:     "json suffix",
			path:     "/api/v1/links.json",
			accept:   "",
			expected: true,
		},
		{
			name:     "json accept header",
			path:     "/api/v1/links",
			accept:   "application/json",
			expected: true,
		},
		{
			name:     "json with charset in accept",
			path:     "/api/v1/links",
			accept:   "application/json; charset=utf-8",
			expected: true,
		},
		{
			name:     "no preference",
			path:     "/api/v1/links",
			accept:   "",
			expected: false,
		},
		{
			name:     "wildcard accept",
			path:     "/api/v1/links",
			accept:   "*/*",
			expected: false,
		},
		{
			name:     "text html accept",
			path:     "/api/v1/links",
			accept:   "text/html",
			expected: false,
		},
		{
			name:     "json suffix with query string",
			path:     "/api/v1/links.json?limit=10",
			accept:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			result := wantsJSON(req)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestWantsPlainText(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		accept   string
		expected bool
	}{
		{
			name:     "txt suffix",
			path:     "/api/v1/links/123.txt",
			accept:   "",
			expected: true,
		},
		{
			name:     "text plain accept header",
			path:     "/api/v1/links/123",
			accept:   "text/plain",
			expected: true,
		},
		{
			name:     "no preference",
			path:     "/api/v1/links/123",
			accept:   "",
			expected: false,
		},
		{
			name:     "json accept",
			path:     "/api/v1/links/123",
			accept:   "application/json",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			result := wantsPlainText(req)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTrimFormatSuffix(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "json suffix",
			path:     "/api/v1/links.json",
			expected: "/api/v1/links",
		},
		{
			name:     "txt suffix",
			path:     "/api/v1/links/123.txt",
			expected: "/api/v1/links/123",
		},
		{
			name:     "no suffix",
			path:     "/api/v1/links",
			expected: "/api/v1/links",
		},
		{
			name:     "json in middle of path",
			path:     "/api/v1/links.json/extra",
			expected: "/api/v1/links.json/extra",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "just suffix",
			path:     ".json",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimFormatSuffix(tt.path)

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsAuthorizedAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		apiKey     string
		secret     string
		expected   bool
	}{
		{
			name:       "valid header matches secret",
			remoteAddr: "192.168.1.100:12345",
			apiKey:     "my-secret-key",
			secret:     "my-secret-key",
			expected:   true,
		},
		{
			name:       "invalid header does not match",
			remoteAddr: "192.168.1.100:12345",
			apiKey:     "wrong-key",
			secret:     "my-secret-key",
			expected:   false,
		},
		{
			name:       "missing header",
			remoteAddr: "192.168.1.100:12345",
			apiKey:     "",
			secret:     "my-secret-key",
			expected:   false,
		},
		{
			name:       "no secret configured",
			remoteAddr: "192.168.1.100:12345",
			apiKey:     "some-key",
			secret:     "",
			expected:   false,
		},
		{
			name:       "localhost IPv4 always allowed",
			remoteAddr: "127.0.0.1:12345",
			apiKey:     "",
			secret:     "my-secret-key",
			expected:   true,
		},
		{
			name:       "localhost IPv6 always allowed",
			remoteAddr: "[::1]:12345",
			apiKey:     "",
			secret:     "my-secret-key",
			expected:   true,
		},
		{
			name:       "localhost without port",
			remoteAddr: "127.0.0.1",
			apiKey:     "",
			secret:     "my-secret-key",
			expected:   true,
		},
		{
			name:       "localhost string always allowed",
			remoteAddr: "localhost:8080",
			apiKey:     "",
			secret:     "my-secret-key",
			expected:   true,
		},
		{
			name:       "localhost allowed even without secret",
			remoteAddr: "127.0.0.1:12345",
			apiKey:     "",
			secret:     "",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			result := isAuthorizedAPIKey(req, tt.secret)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
