package checker

import (
	"testing"
)

func TestParseTarget(t *testing.T) {
	testCases := []struct {
		name        string
		target      string
		wantScheme  string
		wantHost    string
		wantPort    string
		wantPath    string
		wantFullURL string
	}{
		{
			name:        "Simple domain",
			target:      "example.com",
			wantScheme:  "http",
			wantHost:    "example.com",
			wantPort:    "",
			wantPath:    "",
			wantFullURL: "http://example.com",
		},
		{
			name:        "HTTP URL",
			target:      "http://example.com",
			wantScheme:  "http",
			wantHost:    "example.com",
			wantPort:    "",
			wantPath:    "",
			wantFullURL: "http://example.com",
		},
		{
			name:        "HTTPS URL",
			target:      "https://example.com",
			wantScheme:  "https",
			wantHost:    "example.com",
			wantPort:    "",
			wantPath:    "",
			wantFullURL: "https://example.com",
		},
		{
			name:        "URL with port",
			target:      "https://example.com:8443",
			wantScheme:  "https",
			wantHost:    "example.com",
			wantPort:    "8443",
			wantPath:    "",
			wantFullURL: "https://example.com:8443",
		},
		{
			name:        "Domain with port",
			target:      "example.com:8080",
			wantScheme:  "http",
			wantHost:    "example.com",
			wantPort:    "8080",
			wantPath:    "",
			wantFullURL: "http://example.com:8080",
		},
		{
			name:        "URL with path",
			target:      "https://example.com/api/v1",
			wantScheme:  "https",
			wantHost:    "example.com",
			wantPort:    "",
			wantPath:    "/api/v1",
			wantFullURL: "https://example.com/api/v1",
		},
		{
			name:        "URL with port and path",
			target:      "https://example.com:443/path",
			wantScheme:  "https",
			wantHost:    "example.com",
			wantPort:    "443",
			wantPath:    "/path",
			wantFullURL: "https://example.com:443/path",
		},
		{
			name:        "Subdomain",
			target:      "api.example.com",
			wantScheme:  "http",
			wantHost:    "api.example.com",
			wantPort:    "",
			wantPath:    "",
			wantFullURL: "http://api.example.com",
		},
		{
			name:        "Domain with trailing slash",
			target:      "example.com/",
			wantScheme:  "http",
			wantHost:    "example.com",
			wantPort:    "",
			wantPath:    "/",
			wantFullURL: "http://example.com/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := ParseTarget(tc.target)

			if info.Original != tc.target {
				t.Errorf("Original: expected '%s', got '%s'", tc.target, info.Original)
			}

			if info.Scheme != tc.wantScheme {
				t.Errorf("Scheme: expected '%s', got '%s'", tc.wantScheme, info.Scheme)
			}

			if info.Host != tc.wantHost {
				t.Errorf("Host: expected '%s', got '%s'", tc.wantHost, info.Host)
			}

			if info.Port != tc.wantPort {
				t.Errorf("Port: expected '%s', got '%s'", tc.wantPort, info.Port)
			}

			if info.Path != tc.wantPath {
				t.Errorf("Path: expected '%s', got '%s'", tc.wantPath, info.Path)
			}

			if info.FullURL != tc.wantFullURL {
				t.Errorf("FullURL: expected '%s', got '%s'", tc.wantFullURL, info.FullURL)
			}
		})
	}
}

func TestNormalizeHTTPTarget(t *testing.T) {
	testCases := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "Simple domain",
			target:   "example.com",
			expected: "http://example.com",
		},
		{
			name:     "Already has HTTP",
			target:   "http://example.com",
			expected: "http://example.com",
		},
		{
			name:     "Already has HTTPS",
			target:   "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "Domain with port",
			target:   "example.com:8080",
			expected: "http://example.com:8080",
		},
		{
			name:     "URL with path",
			target:   "https://example.com/api",
			expected: "https://example.com/api",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := NormalizeHTTPTarget(tc.target)
			if normalized != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, normalized)
			}
		})
	}
}

func TestExtractHost(t *testing.T) {
	testCases := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "Simple domain",
			target:   "example.com",
			expected: "example.com",
		},
		{
			name:     "HTTP URL",
			target:   "http://example.com",
			expected: "example.com",
		},
		{
			name:     "HTTPS URL",
			target:   "https://example.com",
			expected: "example.com",
		},
		{
			name:     "URL with port",
			target:   "https://example.com:443",
			expected: "example.com",
		},
		{
			name:     "Domain with port",
			target:   "example.com:8080",
			expected: "example.com",
		},
		{
			name:     "URL with path",
			target:   "https://example.com/api/v1",
			expected: "example.com",
		},
		{
			name:     "URL with port and path",
			target:   "https://example.com:443/path",
			expected: "example.com",
		},
		{
			name:     "Subdomain",
			target:   "api.example.com",
			expected: "api.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			host := ExtractHost(tc.target)
			if host != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, host)
			}
		})
	}
}

func TestParseTarget_IPv4(t *testing.T) {
	testCases := []struct {
		name     string
		target   string
		wantHost string
	}{
		{
			name:     "Plain IPv4",
			target:   "192.168.1.1",
			wantHost: "192.168.1.1",
		},
		{
			name:     "IPv4 with HTTP",
			target:   "http://192.168.1.1",
			wantHost: "192.168.1.1",
		},
		{
			name:     "IPv4 with port",
			target:   "192.168.1.1:8080",
			wantHost: "192.168.1.1",
		},
		{
			name:     "IPv4 with HTTPS and port",
			target:   "https://192.168.1.1:443",
			wantHost: "192.168.1.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := ParseTarget(tc.target)
			if info.Host != tc.wantHost {
				t.Errorf("Expected host '%s', got '%s'", tc.wantHost, info.Host)
			}
		})
	}
}

func TestParseTarget_IPv6(t *testing.T) {
	testCases := []struct {
		name     string
		target   string
		wantHost string
	}{
		{
			name:     "IPv6 with brackets",
			target:   "http://[2001:db8::1]",
			wantHost: "2001:db8::1",
		},
		{
			name:     "IPv6 with brackets and port",
			target:   "https://[2001:db8::1]:443",
			wantHost: "2001:db8::1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := ParseTarget(tc.target)
			if info.Host != tc.wantHost {
				t.Errorf("Expected host '%s', got '%s'", tc.wantHost, info.Host)
			}
		})
	}
}

func TestParseTarget_EdgeCases(t *testing.T) {
	testCases := []struct {
		name   string
		target string
	}{
		{
			name:   "Empty string",
			target: "",
		},
		{
			name:   "Just protocol",
			target: "http://",
		},
		{
			name:   "Malformed URL",
			target: "ht!tp://example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			info := ParseTarget(tc.target)
			if info == nil {
				t.Fatalf("Expected non-nil result for target %q", tc.target)
			}
			if info.Original != tc.target {
				t.Errorf("Expected original '%s', got '%s'", tc.target, info.Original)
			}
		})
	}
}
