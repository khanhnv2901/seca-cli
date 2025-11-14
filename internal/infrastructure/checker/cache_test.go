package checker

import (
	"net/http"
	"testing"
)

func TestAnalyzeCachePolicy_NoHeaders(t *testing.T) {
	policy := AnalyzeCachePolicy(http.Header{})
	if policy == nil {
		t.Fatal("expected policy")
	}
	if len(policy.Issues) == 0 {
		t.Error("expected issue for missing headers")
	}
}

func TestAnalyzeCachePolicy_WithHeaders(t *testing.T) {
	hdr := http.Header{}
	hdr.Set("Cache-Control", "public, max-age=3600")
	hdr.Set("Expires", "Tue, 15 Nov 2025 12:45:26 GMT")

	policy := AnalyzeCachePolicy(hdr)
	if policy == nil {
		t.Fatal("expected policy")
	}
	if len(policy.Issues) != 0 {
		t.Fatalf("expected no issues, got %v", policy.Issues)
	}
	if policy.CacheControl != "public, max-age=3600" {
		t.Errorf("unexpected cache-control value: %s", policy.CacheControl)
	}
}
