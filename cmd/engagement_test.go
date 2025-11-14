package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEngagementService_ListEmpty(t *testing.T) {
	defer setupTestAppContextWithServices(t)()

	ctx := context.Background()
	engagements, err := globalAppContext.Services.EngagementService.ListEngagements(ctx)
	if err != nil {
		t.Fatalf("ListEngagements() error = %v", err)
	}
	if len(engagements) != 0 {
		t.Fatalf("expected empty engagement list, got %d", len(engagements))
	}
}

func TestEngagementService_CreateAndRetrieve(t *testing.T) {
	defer setupTestAppContextWithServices(t)()

	ctx := context.Background()
	svc := globalAppContext.Services.EngagementService

	created, err := svc.CreateEngagement(ctx, "Test Engagement", "owner@example.com", "Test ROE", []string{"https://example.com"})
	if err != nil {
		t.Fatalf("CreateEngagement() error = %v", err)
	}
	if err := svc.AcknowledgeROE(ctx, created.ID()); err != nil {
		t.Fatalf("AcknowledgeROE() error = %v", err)
	}

	fetched, err := svc.GetEngagement(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetEngagement() error = %v", err)
	}

	if fetched.Name() != "Test Engagement" {
		t.Errorf("expected name to be %q, got %q", "Test Engagement", fetched.Name())
	}
	if fetched.Owner() != "owner@example.com" {
		t.Errorf("expected owner to be %q, got %q", "owner@example.com", fetched.Owner())
	}
	if !fetched.ROEAgreed() {
		t.Error("expected ROE to be acknowledged")
	}
	if len(fetched.Scope()) != 1 || fetched.Scope()[0] != "https://example.com" {
		t.Errorf("expected initial scope to include https://example.com, got %+v", fetched.Scope())
	}
}

func TestEngagementService_AddScopeEntries(t *testing.T) {
	defer setupTestAppContextWithServices(t)()

	ctx := context.Background()
	svc := globalAppContext.Services.EngagementService

	created, err := svc.CreateEngagement(ctx, "Scope Test", "owner@example.com", "Test ROE", nil)
	if err != nil {
		t.Fatalf("CreateEngagement() error = %v", err)
	}

	normalized, err := normalizeScopeEntries(created.ID(), []string{" https://api.example.com "})
	if err != nil {
		t.Fatalf("normalizeScopeEntries() error = %v", err)
	}

	if err := svc.AddToScope(ctx, created.ID(), normalized); err != nil {
		t.Fatalf("AddToScope() error = %v", err)
	}

	fetched, err := svc.GetEngagement(ctx, created.ID())
	if err != nil {
		t.Fatalf("GetEngagement() error = %v", err)
	}

	if len(fetched.Scope()) != 1 || fetched.Scope()[0] != "https://api.example.com" {
		t.Fatalf("expected scope entry to be normalized, got %+v", fetched.Scope())
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
