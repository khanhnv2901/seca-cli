package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/khanhnv2901/seca-cli/cmd/testutil"
	consts "github.com/khanhnv2901/seca-cli/internal/constants"
)

// Helper function to backup and restore engagements file
// Note: This uses testutil.SetupEngagementsFile internally
func setupTestEngagements(t *testing.T) func() {
	return testutil.SetupEngagementsFile(t, getEngagementsFilePath)
}

func TestLoadEngagements_EmptyFile(t *testing.T) {
	cleanup := setupTestEngagements(t)
	defer cleanup()

	// Test loading when file doesn't exist
	result := loadEngagements()
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d engagements", len(result))
	}
}

func TestLoadEngagements_ValidFile(t *testing.T) {
	cleanup := setupTestEngagements(t)
	defer cleanup()

	// Create test data
	testEngagements := []Engagement{
		{
			ID:        "123456",
			Name:      "Test Engagement",
			Owner:     "test@example.com",
			ROE:       "Test ROE",
			ROEAgree:  true,
			CreatedAt: time.Now(),
			Scope:     []string{"https://example.com"},
		},
	}

	// Get the actual file path
	filePath, err := getEngagementsFilePath()
	if err != nil {
		t.Fatalf("Failed to get engagements file path: %v", err)
	}

	// Write test data
	data, _ := json.MarshalIndent(testEngagements, jsonPrefix, jsonIndent)
	if err := os.WriteFile(filePath, data, consts.DefaultFilePerm); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load and verify
	result := loadEngagements()
	if len(result) != 1 {
		t.Fatalf("Expected 1 engagement, got %d", len(result))
	}

	if result[0].ID != "123456" {
		t.Errorf("Expected ID '123456', got '%s'", result[0].ID)
	}

	if result[0].Name != "Test Engagement" {
		t.Errorf("Expected name 'Test Engagement', got '%s'", result[0].Name)
	}

	if result[0].Owner != "test@example.com" {
		t.Errorf("Expected owner 'test@example.com', got '%s'", result[0].Owner)
	}

	if !result[0].ROEAgree {
		t.Error("Expected ROEAgree to be true")
	}

	if len(result[0].Scope) != 1 {
		t.Errorf("Expected 1 scope item, got %d", len(result[0].Scope))
	}
}

func TestSaveEngagements(t *testing.T) {
	cleanup := setupTestEngagements(t)
	defer cleanup()

	testEngagements := []Engagement{
		{
			ID:        "789012",
			Name:      "Save Test",
			Owner:     "save@example.com",
			ROE:       "Test ROE",
			ROEAgree:  true,
			CreatedAt: time.Now(),
			Scope:     []string{"https://test.com"},
		},
	}

	// Save engagements
	saveEngagements(testEngagements)

	// Get the actual file path (which might be in data directory)
	filePath, err := getEngagementsFilePath()
	if err != nil {
		t.Fatalf("Failed to get engagements file path: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("File was not created at %s", filePath)
	}

	// Load back and verify
	data, _ := os.ReadFile(filePath)
	var loaded []Engagement
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 engagement, got %d", len(loaded))
	}

	if loaded[0].ID != "789012" {
		t.Errorf("Expected ID '789012', got '%s'", loaded[0].ID)
	}
}

func TestEngagement_ValidData(t *testing.T) {
	engagement := Engagement{
		ID:        "123",
		Name:      "Test",
		Owner:     "owner@example.com",
		Start:     time.Now(),
		End:       time.Now().Add(24 * time.Hour),
		Scope:     []string{"https://example.com", "https://api.example.com"},
		ROE:       "Test ROE text",
		ROEAgree:  true,
		CreatedAt: time.Now(),
	}

	if engagement.ID != "123" {
		t.Errorf("Expected ID '123', got '%s'", engagement.ID)
	}

	if len(engagement.Scope) != 2 {
		t.Errorf("Expected 2 scope items, got %d", len(engagement.Scope))
	}

	if !engagement.ROEAgree {
		t.Error("Expected ROEAgree to be true")
	}
}

func TestEngagement_JSONMarshaling(t *testing.T) {
	engagement := Engagement{
		ID:        "456",
		Name:      "JSON Test",
		Owner:     "json@example.com",
		ROE:       "Test",
		ROEAgree:  true,
		CreatedAt: time.Now(),
		Scope:     []string{"https://example.com"},
	}

	// Marshal to JSON
	data, err := json.Marshal(engagement)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded Engagement
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != engagement.ID {
		t.Errorf("ID mismatch: expected '%s', got '%s'", engagement.ID, decoded.ID)
	}

	if decoded.Name != engagement.Name {
		t.Errorf("Name mismatch: expected '%s', got '%s'", engagement.Name, decoded.Name)
	}
}

func TestEngagement_MultipleEngagements(t *testing.T) {
	cleanup := setupTestEngagements(t)
	defer cleanup()

	engagements := []Engagement{
		{
			ID:        "1",
			Name:      "First",
			Owner:     "first@example.com",
			ROEAgree:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:        "2",
			Name:      "Second",
			Owner:     "second@example.com",
			ROEAgree:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:        "3",
			Name:      "Third",
			Owner:     "third@example.com",
			ROEAgree:  true,
			CreatedAt: time.Now(),
		},
	}

	saveEngagements(engagements)
	loaded := loadEngagements()

	if len(loaded) != 3 {
		t.Fatalf("Expected 3 engagements, got %d", len(loaded))
	}

	for i, eng := range loaded {
		if eng.ID != engagements[i].ID {
			t.Errorf("Engagement %d: expected ID '%s', got '%s'", i, engagements[i].ID, eng.ID)
		}
	}
}

func TestValidateURLScope(t *testing.T) {
	valid := []string{
		"https://example.com",
		"http://example.com/login",
		"https://127.0.0.1",
	}
	for _, raw := range valid {
		if err := validateURLScope(raw); err != nil {
			t.Fatalf("expected %s to be valid, got %v", raw, err)
		}
	}

	invalid := []string{
		"ftp://example.com",
		"https://",
		"http://?foo=bar",
		"https://exa mple.com",
	}
	for _, raw := range invalid {
		if err := validateURLScope(raw); err == nil {
			t.Fatalf("expected %s to be invalid", raw)
		}
	}
}

func TestIsValidHostname(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.domain-example.com",
		"localhost",
		"xn--bcher-kva.example",
	}
	for _, host := range valid {
		if !isValidHostname(host) {
			t.Fatalf("expected %s to be valid", host)
		}
	}

	invalid := []string{
		"",
		"-bad.example.com",
		"bad-.example.com",
		"toolong" + strings.Repeat("a", 250) + ".com",
		"with space.com",
		"bad_label!.com",
	}
	for _, host := range invalid {
		if isValidHostname(host) {
			t.Fatalf("expected %s to be invalid", host)
		}
	}
}

func TestIsValidHostOrIP(t *testing.T) {
	if !isValidHostOrIP("10.0.0.1") {
		t.Fatal("expected IPv4 to be valid")
	}
	if !isValidHostOrIP("example.com") {
		t.Fatal("expected hostname to be valid")
	}
	if isValidHostOrIP("") {
		t.Fatal("expected empty host to be invalid")
	}
}

func TestValidateScopeEntry(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.domain.com",
		"192.168.0.1",
	}
	for _, entry := range valid {
		if err := validateScopeEntry(entry); err != nil {
			t.Fatalf("expected %s to be valid, got %v", entry, err)
		}
	}

	invalid := []string{
		"http://", // missing host
		"bad host",
		"ftp://example.com",
	}
	for _, entry := range invalid {
		if err := validateScopeEntry(entry); err == nil {
			t.Fatalf("expected %s to be invalid", entry)
		}
	}
}

func TestFindEngagementByID(t *testing.T) {
	cleanup := setupTestEngagements(t)
	defer cleanup()

	list := []Engagement{
		{ID: "abc", Name: "First"},
		{ID: "xyz", Name: "Second"},
	}
	saveEngagements(list)

	eng, err := findEngagementByID("xyz")
	if err != nil {
		t.Fatalf("expected engagement to be found, got %v", err)
	}
	if eng.Name != "Second" {
		t.Fatalf("expected Second, got %s", eng.Name)
	}

	if _, err := findEngagementByID(""); err == nil {
		t.Fatal("expected error for missing id")
	}

	if _, err := findEngagementByID("missing"); err == nil {
		t.Fatal("expected error for missing engagement")
	}
}

func TestEngagement_ScopeHandling(t *testing.T) {
	cleanup := setupTestEngagements(t)
	defer cleanup()

	engagement := Engagement{
		ID:        "scope-test",
		Name:      "Scope Test",
		Owner:     "test@example.com",
		ROEAgree:  true,
		CreatedAt: time.Now(),
		Scope:     []string{},
	}

	// Test with empty scope
	if len(engagement.Scope) != 0 {
		t.Errorf("Expected empty scope, got %d items", len(engagement.Scope))
	}

	// Add scope items
	engagement.Scope = append(engagement.Scope, "https://example.com")
	engagement.Scope = append(engagement.Scope, "https://api.example.com")

	if len(engagement.Scope) != 2 {
		t.Errorf("Expected 2 scope items, got %d", len(engagement.Scope))
	}

	// Save and reload
	saveEngagements([]Engagement{engagement})
	loaded := loadEngagements()

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 engagement, got %d", len(loaded))
	}

	if len(loaded[0].Scope) != 2 {
		t.Errorf("Expected 2 scope items after reload, got %d", len(loaded[0].Scope))
	}
}

func TestEngagement_ROEAgreeValidation(t *testing.T) {
	engagement := Engagement{
		ID:        "roe-test",
		Name:      "ROE Test",
		Owner:     "test@example.com",
		ROE:       "Test ROE",
		ROEAgree:  false,
		CreatedAt: time.Now(),
	}

	if engagement.ROEAgree {
		t.Error("Expected ROEAgree to be false")
	}

	engagement.ROEAgree = true

	if !engagement.ROEAgree {
		t.Error("Expected ROEAgree to be true after update")
	}
}

func TestNormalizeScopeEntries(t *testing.T) {
	t.Run("valid entries", func(t *testing.T) {
		input := []string{
			" https://example.com/login ",
			"api.example.com",
			"example.com:8443/report",
			"192.168.1.10",
		}
		normalized, err := normalizeScopeEntries("eng-123", input)
		if err != nil {
			t.Fatalf("Expected entries to be valid, got error: %v", err)
		}
		for i, original := range input {
			if normalized[i] != strings.TrimSpace(original) {
				t.Errorf("Entry %d was not trimmed correctly. Expected %q, got %q", i, strings.TrimSpace(original), normalized[i])
			}
		}
	})

	t.Run("invalid entries", func(t *testing.T) {
		testCases := [][]string{
			{""},
			{"ftp://example.com"},
			{"exa mple.com"},
			{"http://"},
		}

		for _, tc := range testCases {
			if _, err := normalizeScopeEntries("eng-123", tc); err == nil {
				t.Errorf("Expected error for scope entries %v, but got none", tc)
			}
		}
	})
}
