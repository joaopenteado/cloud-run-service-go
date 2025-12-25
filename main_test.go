package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := `{"status":"healthy"}`
	if body := w.Body.String(); body != expected {
		t.Errorf("expected body %q, got %q", expected, body)
	}
}

func TestReadinessHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/readiness", nil)
	w := httptest.NewRecorder()

	readinessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := `{"status":"ready"}`
	if body := w.Body.String(); body != expected {
		t.Errorf("expected body %q, got %q", expected, body)
	}
}

func TestLivenessHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/liveness", nil)
	w := httptest.NewRecorder()

	livenessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := `{"status":"alive"}`
	if body := w.Body.String(); body != expected {
		t.Errorf("expected body %q, got %q", expected, body)
	}
}

func TestRootHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	rootHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Cloud Run Service Go") {
		t.Errorf("expected body to contain 'Cloud Run Service Go', got %q", body)
	}
}

func TestHelloHandler(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "no name parameter",
			query:    "",
			expected: `{"message":"Hello, World!"}`,
		},
		{
			name:     "with name parameter",
			query:    "?name=John",
			expected: `{"message":"Hello, John!"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/hello"+tt.query, nil)
			w := httptest.NewRecorder()

			helloHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			if body := w.Body.String(); body != tt.expected {
				t.Errorf("expected body %q, got %q", tt.expected, body)
			}
		})
	}
}

func TestEchoHandler(t *testing.T) {
	body := `{"test":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/echo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	echoHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if responseBody := w.Body.String(); responseBody != body {
		t.Errorf("expected body %q, got %q", body, responseBody)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", ct)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "returns default when env not set",
			key:          "NON_EXISTENT_KEY",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
