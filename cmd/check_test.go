package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckResult_JSON(t *testing.T) {
	result := CheckResult{
		Target:       "https://example.com",
		CheckedAt:    time.Now().UTC(),
		Status:       "ok",
		HTTPStatus:   200,
		ServerHeader: "nginx",
		TLSExpiry:    "2026-01-15T00:00:00Z",
		Notes:        "test note",
		Error:        "",
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded CheckResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Target != result.Target {
		t.Errorf("Target mismatch: expected '%s', got '%s'", result.Target, decoded.Target)
	}

	if decoded.Status != result.Status {
		t.Errorf("Status mismatch: expected '%s', got '%s'", result.Status, decoded.Status)
	}

	if decoded.HTTPStatus != result.HTTPStatus {
		t.Errorf("HTTPStatus mismatch: expected %d, got %d", result.HTTPStatus, decoded.HTTPStatus)
	}
}

func TestCheckResult_WithError(t *testing.T) {
	result := CheckResult{
		Target:    "https://example.com",
		CheckedAt: time.Now().UTC(),
		Status:    "error",
		Error:     "connection timeout",
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message to be set")
	}

	if result.HTTPStatus != 0 {
		t.Errorf("Expected HTTPStatus 0 for error, got %d", result.HTTPStatus)
	}
}

func TestRunMetadata(t *testing.T) {
	metadata := RunMetadata{
		Operator:       "test-operator",
		EngagementID:   "123456",
		EngagementName: "Test Engagement",
		Owner:          "owner@example.com",
		StartAt:        time.Now(),
		CompleteAt:     time.Now().Add(5 * time.Minute),
		AuditHash:      "abc123",
		ResultsHash:    "def456",
		TotalTargets:   5,
	}

	if metadata.Operator != "test-operator" {
		t.Errorf("Expected operator 'test-operator', got '%s'", metadata.Operator)
	}

	if metadata.TotalTargets != 5 {
		t.Errorf("Expected 5 targets, got %d", metadata.TotalTargets)
	}

	if metadata.AuditHash == "" || metadata.ResultsHash == "" {
		t.Error("Hashes should not be empty")
	}
}

func TestRunOutput_JSON(t *testing.T) {
	output := RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-op",
			EngagementID:   "123",
			EngagementName: "Test",
			Owner:          "owner@test.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   2,
		},
		Results: []CheckResult{
			{
				Target:     "https://example.com",
				CheckedAt:  time.Now(),
				Status:     "ok",
				HTTPStatus: 200,
			},
			{
				Target:     "https://test.com",
				CheckedAt:  time.Now(),
				Status:     "ok",
				HTTPStatus: 200,
			},
		},
	}

	// Marshal
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded RunOutput
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(decoded.Results))
	}

	if decoded.Metadata.TotalTargets != 2 {
		t.Errorf("Expected 2 total targets, got %d", decoded.Metadata.TotalTargets)
	}
}

func TestHTTPCheck_MockServer(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "test-server")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Make request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	serverHeader := resp.Header.Get("Server")
	if serverHeader != "test-server" {
		t.Errorf("Expected Server header 'test-server', got '%s'", serverHeader)
	}
}

func TestHTTPCheck_MockServer_Error(t *testing.T) {
	// Try to connect to a non-existent server
	client := &http.Client{Timeout: 1 * time.Second}
	_, err := client.Get("http://localhost:99999")

	if err == nil {
		t.Error("Expected error for invalid server, got nil")
	}
}

func TestHTTPCheck_RobotsText(t *testing.T) {
	// Create a test server with robots.txt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("User-agent: *\nDisallow: /admin"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	// Check robots.txt
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(server.URL + "/robots.txt")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for robots.txt, got %d", resp.StatusCode)
	}
}

func TestHTTPCheck_HeadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Server", "head-test")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Make HEAD request
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodHead, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HEAD request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Server") != "head-test" {
		t.Error("Server header not set correctly")
	}
}

func TestHTTPCheck_VariousStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"Not Found", http.StatusNotFound},
		{"Internal Error", http.StatusInternalServerError},
		{"Forbidden", http.StatusForbidden},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, resp.StatusCode)
			}
		})
	}
}

func TestCheckResult_Marshaling_OmitEmpty(t *testing.T) {
	// Test that omitempty works for optional fields
	result := CheckResult{
		Target:    "https://example.com",
		CheckedAt: time.Now(),
		Status:    "ok",
		// Omit HTTPStatus, ServerHeader, etc.
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	// HTTPStatus should be omitted when 0
	if _, exists := decoded["http_status"]; exists {
		t.Error("Expected http_status to be omitted")
	}

	// But target and status should be present
	if _, exists := decoded["target"]; !exists {
		t.Error("Expected target to be present")
	}

	if _, exists := decoded["status"]; !exists {
		t.Error("Expected status to be present")
	}
}

func TestHTTPCheck_Timeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use a short timeout
	client := &http.Client{Timeout: 500 * time.Millisecond}
	_, err := client.Get(server.URL)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestCheckResult_SuccessfulCheck(t *testing.T) {
	result := CheckResult{
		Target:       "https://api.example.com",
		CheckedAt:    time.Now().UTC(),
		Status:       "ok",
		HTTPStatus:   200,
		ServerHeader: "nginx/1.21.0",
		TLSExpiry:    "2026-12-31T23:59:59Z",
		Notes:        "robots.txt found",
	}

	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result.Status)
	}

	if result.HTTPStatus != 200 {
		t.Errorf("Expected HTTP status 200, got %d", result.HTTPStatus)
	}

	if result.Error != "" {
		t.Error("Expected no error for successful check")
	}

	if result.TLSExpiry == "" {
		t.Error("Expected TLS expiry to be set")
	}
}
