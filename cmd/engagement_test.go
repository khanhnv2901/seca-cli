package cmd

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// Helper function to backup and restore engagements file
func setupTestEngagements(t *testing.T) func() {
	t.Helper()

	// Backup existing engagements.json if it exists
	backupFile := "engagements.json.backup"
	if _, err := os.Stat(engagementsFile); err == nil {
		data, _ := os.ReadFile(engagementsFile)
		_ = os.WriteFile(backupFile, data, 0644)
	}

	// Remove existing file
	os.Remove(engagementsFile)

	// Return cleanup function
	return func() {
		// Restore backup if it existed
		if _, err := os.Stat(backupFile); err == nil {
			data, _ := os.ReadFile(backupFile)
			_ = os.WriteFile(engagementsFile, data, 0644)
			_ = os.Remove(backupFile)
		} else {
			// Just remove test file
			_ = os.Remove(engagementsFile)
		}
	}
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

	// Write test data
	data, _ := json.MarshalIndent(testEngagements, "", "  ")
	os.WriteFile(engagementsFile, data, 0644)

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

	// Verify file exists
	if _, err := os.Stat(engagementsFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Load back and verify
	data, _ := os.ReadFile(engagementsFile)
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
