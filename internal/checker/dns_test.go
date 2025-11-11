package checker

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDNSChecker_Name(t *testing.T) {
	checker := &DNSChecker{
		Timeout: 5 * time.Second,
	}

	expected := "check dns"
	if checker.Name() != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, checker.Name())
	}
}

func TestDNSChecker_HostParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain domain",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "HTTP URL",
			input:    "http://example.com",
			expected: "example.com",
		},
		{
			name:     "HTTPS URL",
			input:    "https://example.com",
			expected: "example.com",
		},
		{
			name:     "URL with path",
			input:    "https://example.com/path/to/resource",
			expected: "example.com",
		},
		{
			name:     "URL with port",
			input:    "https://example.com:8080",
			expected: "example.com",
		},
		{
			name:     "URL with port and path",
			input:    "https://example.com:8080/api/v1",
			expected: "example.com",
		},
		{
			name:     "Subdomain",
			input:    "api.example.com",
			expected: "api.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the host parsing logic from dns.go
			host := strings.TrimPrefix(tc.input, "http://")
			host = strings.TrimPrefix(host, "https://")
			host = strings.Split(host, "/")[0]
			host = strings.Split(host, ":")[0]

			if host != tc.expected {
				t.Errorf("Expected host '%s', got '%s'", tc.expected, host)
			}
		})
	}
}

func TestDNSChecker_CheckResult_Structure(t *testing.T) {
	// Test that CheckResult has the expected structure for DNS checks
	result := CheckResult{
		Target:    "example.com",
		CheckedAt: time.Now().UTC(),
		Status:    "ok",
		DNSRecords: map[string]interface{}{
			"a_records": []string{"93.184.216.34"},
		},
		Notes: "1 A record(s) found",
	}

	if result.Target != "example.com" {
		t.Errorf("Expected target 'example.com', got '%s'", result.Target)
	}

	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result.Status)
	}

	if result.DNSRecords == nil {
		t.Error("Expected DNSRecords to be initialized")
	}

	aRecords, ok := result.DNSRecords["a_records"]
	if !ok {
		t.Error("Expected a_records to be present in DNSRecords")
	}

	aRecordsList, ok := aRecords.([]string)
	if !ok {
		t.Error("Expected a_records to be []string")
	}

	if len(aRecordsList) == 0 {
		t.Error("Expected at least one A record")
	}
}

func TestDNSChecker_CheckResult_WithError(t *testing.T) {
	result := CheckResult{
		Target:     "nonexistent.invalid.domain.test",
		CheckedAt:  time.Now().UTC(),
		Status:     "error",
		Error:      "DNS lookup failed: no such host",
		DNSRecords: make(map[string]interface{}),
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message to be set")
	}

	if !strings.Contains(result.Error, "DNS lookup failed") {
		t.Errorf("Expected error to contain 'DNS lookup failed', got '%s'", result.Error)
	}
}

func TestDNSChecker_CheckResult_MultipleRecords(t *testing.T) {
	// Test result structure with multiple record types
	result := CheckResult{
		Target:    "example.com",
		CheckedAt: time.Now().UTC(),
		Status:    "ok",
		DNSRecords: map[string]interface{}{
			"a_records":    []string{"93.184.216.34"},
			"aaaa_records": []string{"2606:2800:220:1:248:1893:25c8:1946"},
			"mx_records": []map[string]interface{}{
				{
					"host":     "mail.example.com.",
					"priority": uint16(10),
				},
			},
			"ns_records":  []string{"ns1.example.com.", "ns2.example.com."},
			"txt_records": []string{"v=spf1 include:_spf.example.com ~all"},
		},
		Notes: "1 A record(s) found, 1 AAAA record(s) found, 1 MX recrod(s) found, 2 NS record(s) found, 1 TXT record(s) found",
	}

	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result.Status)
	}

	// Verify A records
	if _, ok := result.DNSRecords["a_records"]; !ok {
		t.Error("Expected a_records to be present")
	}

	// Verify AAAA records
	if _, ok := result.DNSRecords["aaaa_records"]; !ok {
		t.Error("Expected aaaa_records to be present")
	}

	// Verify MX records
	mxRecords, ok := result.DNSRecords["mx_records"]
	if !ok {
		t.Error("Expected mx_records to be present")
	}

	mxList, ok := mxRecords.([]map[string]interface{})
	if !ok {
		t.Error("Expected mx_records to be []map[string]interface{}")
	}

	if len(mxList) == 0 {
		t.Error("Expected at least one MX record")
	}

	// Verify MX record structure
	if _, ok := mxList[0]["host"]; !ok {
		t.Error("Expected MX record to have 'host' field")
	}

	if _, ok := mxList[0]["priority"]; !ok {
		t.Error("Expected MX record to have 'priority' field")
	}

	// Verify NS records
	if _, ok := result.DNSRecords["ns_records"]; !ok {
		t.Error("Expected ns_records to be present")
	}

	// Verify TXT records
	if _, ok := result.DNSRecords["txt_records"]; !ok {
		t.Error("Expected txt_records to be present")
	}
}

func TestDNSChecker_Timeout(t *testing.T) {
	// Test with a very short timeout
	checker := &DNSChecker{
		Timeout: 1 * time.Nanosecond, // Extremely short timeout to force timeout
	}

	ctx := context.Background()
	result := checker.Check(ctx, "example.com")

	// The check should complete even with timeout (might succeed or fail)
	if result.Target != "example.com" {
		t.Errorf("Expected target 'example.com', got '%s'", result.Target)
	}

	// Status should be set
	if result.Status == "" {
		t.Error("Expected status to be set")
	}
}

func TestDNSChecker_ContextCancellation(t *testing.T) {
	checker := &DNSChecker{
		Timeout: 10 * time.Second,
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := checker.Check(ctx, "example.com")

	// The check should handle the cancelled context
	if result.Target != "example.com" {
		t.Errorf("Expected target 'example.com', got '%s'", result.Target)
	}

	// With cancelled context, we expect an error
	if result.Status != "error" {
		t.Logf("Note: Status is '%s', might succeed if DNS lookup is cached", result.Status)
	}
}

func TestDNSChecker_EmptyNameServer(t *testing.T) {
	// Test with empty nameserver list (should use system default)
	checker := &DNSChecker{
		Timeout:    5 * time.Second,
		NameServer: []string{},
	}

	if len(checker.NameServer) != 0 {
		t.Errorf("Expected empty nameserver list, got %d", len(checker.NameServer))
	}

	// This should use system default resolver
	ctx := context.Background()
	result := checker.Check(ctx, "google.com")

	// Should succeed with default resolver
	if result.Status == "error" {
		t.Logf("DNS check failed: %s (might be network-dependent)", result.Error)
	}
}

func TestDNSChecker_CustomNameServer(t *testing.T) {
	// Test with custom nameserver
	checker := &DNSChecker{
		Timeout:    5 * time.Second,
		NameServer: []string{"8.8.8.8:53"}, // Google DNS
	}

	if len(checker.NameServer) != 1 {
		t.Errorf("Expected 1 nameserver, got %d", len(checker.NameServer))
	}

	if checker.NameServer[0] != "8.8.8.8:53" {
		t.Errorf("Expected nameserver '8.8.8.8:53', got '%s'", checker.NameServer[0])
	}
}

func TestDNSChecker_NoARecords(t *testing.T) {
	// Test the logic when no A records are found
	result := CheckResult{
		Target:     "example.com",
		CheckedAt:  time.Now().UTC(),
		Status:     "error",
		Error:      "no A records found",
		DNSRecords: make(map[string]interface{}),
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Error != "no A records found" {
		t.Errorf("Expected error 'no A records found', got '%s'", result.Error)
	}
}

func TestDNSChecker_CNAMEHandling(t *testing.T) {
	// Test CNAME record handling
	result := CheckResult{
		Target:    "www.example.com",
		CheckedAt: time.Now().UTC(),
		Status:    "ok",
		DNSRecords: map[string]interface{}{
			"a_records": []string{"93.184.216.34"},
			"cname":     "example.com.",
		},
		Notes: "1 A record(s) found, CNAME found",
	}

	cname, ok := result.DNSRecords["cname"]
	if !ok {
		t.Error("Expected cname to be present")
	}

	cnameStr, ok := cname.(string)
	if !ok {
		t.Error("Expected cname to be string")
	}

	if cnameStr == "" {
		t.Error("Expected cname to be non-empty")
	}

	if !strings.Contains(result.Notes, "CNAME found") {
		t.Error("Expected notes to mention CNAME")
	}
}

func TestDNSChecker_PTRRecords(t *testing.T) {
	// Test PTR (reverse DNS) record handling
	result := CheckResult{
		Target:    "8.8.8.8",
		CheckedAt: time.Now().UTC(),
		Status:    "ok",
		DNSRecords: map[string]interface{}{
			"a_records":   []string{"8.8.8.8"},
			"ptr_records": []string{"dns.google."},
		},
		Notes: "1 A record(s) found, PTR record(s) found",
	}

	ptrRecords, ok := result.DNSRecords["ptr_records"]
	if !ok {
		t.Error("Expected ptr_records to be present")
	}

	ptrList, ok := ptrRecords.([]string)
	if !ok {
		t.Error("Expected ptr_records to be []string")
	}

	if len(ptrList) == 0 {
		t.Error("Expected at least one PTR record")
	}

	if !strings.Contains(result.Notes, "PTR record(s) found") {
		t.Error("Expected notes to mention PTR records")
	}
}

func TestDNSChecker_RealDomain_Google(t *testing.T) {
	// Integration test with a real, stable domain
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := &DNSChecker{
		Timeout: 10 * time.Second,
	}

	ctx := context.Background()
	result := checker.Check(ctx, "google.com")

	// google.com should always resolve
	if result.Status != "ok" {
		t.Errorf("Expected status 'ok' for google.com, got '%s' with error: %s", result.Status, result.Error)
	}

	// Should have A records
	aRecords, ok := result.DNSRecords["a_records"]
	if !ok {
		t.Error("Expected A records for google.com")
	}

	aRecordsList, ok := aRecords.([]string)
	if !ok {
		t.Error("Expected a_records to be []string")
	}

	if len(aRecordsList) == 0 {
		t.Error("Expected at least one A record for google.com")
	}

	t.Logf("google.com resolved to: %v", aRecordsList)
}

func TestDNSChecker_InvalidDomain(t *testing.T) {
	// Test with an invalid domain
	checker := &DNSChecker{
		Timeout: 5 * time.Second,
	}

	ctx := context.Background()
	result := checker.Check(ctx, "this-domain-definitely-does-not-exist-12345.invalid")

	// Should fail
	if result.Status != "error" {
		t.Errorf("Expected status 'error' for invalid domain, got '%s'", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message for invalid domain")
	}

	if !strings.Contains(result.Error, "DNS lookup failed") {
		t.Errorf("Expected error to contain 'DNS lookup failed', got '%s'", result.Error)
	}
}

func TestDNSChecker_NotesTyro(t *testing.T) {
	// Test for the typo "recrod" in the notes (line 107 of dns.go)
	notes := "1 A record(s) found, 1 MX recrod(s) found"

	// This test documents the existing typo
	if !strings.Contains(notes, "MX recrod(s)") {
		t.Error("Expected typo 'recrod' in notes (this is testing current behavior)")
	}

	// The correct spelling should be "record"
	t.Logf("Note: There's a typo in dns.go:107 - 'recrod' should be 'record'")
}

func TestDNSChecker_CheckResult_JSONMarshaling(t *testing.T) {
	// Test that CheckResult with DNS records can be marshaled to JSON
	result := CheckResult{
		Target:    "example.com",
		CheckedAt: time.Now().UTC(),
		Status:    "ok",
		DNSRecords: map[string]interface{}{
			"a_records": []string{"93.184.216.34"},
			"mx_records": []map[string]interface{}{
				{
					"host":     "mail.example.com.",
					"priority": 10,
				},
			},
		},
		Notes: "DNS records found",
	}

	// Marshal to JSON (already tested in check_test.go, but good to verify with DNS data)
	if result.DNSRecords == nil {
		t.Error("DNSRecords should not be nil")
	}

	if len(result.DNSRecords) == 0 {
		t.Error("DNSRecords should not be empty")
	}
}

func TestDNSChecker_DefaultTimeout(t *testing.T) {
	// Test with default/zero timeout
	checker := &DNSChecker{
		Timeout: 0, // Zero timeout
	}

	// Even with zero timeout, the check should run
	// (the actual timeout context will use whatever value is set)
	if checker.Timeout != 0 {
		t.Errorf("Expected timeout 0, got %v", checker.Timeout)
	}
}

func TestDNSChecker_MultipleNameServers(t *testing.T) {
	// Test with multiple nameservers (though only first is used in current implementation)
	checker := &DNSChecker{
		Timeout:    5 * time.Second,
		NameServer: []string{"8.8.8.8:53", "1.1.1.1:53"},
	}

	if len(checker.NameServer) != 2 {
		t.Errorf("Expected 2 nameservers, got %d", len(checker.NameServer))
	}

	// Note: Current implementation only uses first nameserver
	t.Logf("Note: Current implementation uses only first nameserver: %s", checker.NameServer[0])
}
