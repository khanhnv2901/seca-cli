package cmd

import "testing"

func TestEngagementNotFoundError(t *testing.T) {
	err := &EngagementNotFoundError{ID: "123"}
	if err.Error() != "engagement 123 not found" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}
}

func TestScopeViolationError(t *testing.T) {
	err := &ScopeViolationError{Target: "example.com", Scope: "123"}
	want := "target example.com is not permitted for engagement 123"
	if err.Error() != want {
		t.Fatalf("expected %s, got %s", want, err.Error())
	}

	err = &ScopeViolationError{Scope: "123"}
	want = "scope 123 is invalid or empty"
	if err.Error() != want {
		t.Fatalf("expected %s, got %s", want, err.Error())
	}

	err = &ScopeViolationError{Target: "example.com"}
	want = "target example.com violates scope policy"
	if err.Error() != want {
		t.Fatalf("expected %s, got %s", want, err.Error())
	}
}
