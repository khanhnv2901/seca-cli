package checker

import (
	"net/http"
	"strings"
	"testing"
)

func TestAnalyzeCORS_AllowsAnyOrigin(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Access-Control-Allow-Origin":      []string{"*"},
			"Access-Control-Allow-Credentials": []string{"true"},
		},
	}

	report := AnalyzeCORS(resp)
	if report == nil {
		t.Fatal("expected CORS report")
	}
	if !report.AllowsAnyOrigin {
		t.Error("expected AllowsAnyOrigin to be true")
	}
	if !report.AllowCredentials {
		t.Error("expected AllowCredentials to be true")
	}
	if len(report.Issues) == 0 {
		t.Error("expected issues to be populated")
	}
}

func TestAnalyzeCORS_MissingHeader(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	report := AnalyzeCORS(resp)
	if report == nil || !report.MissingAllowOrigin {
		t.Fatal("expected missing header issue")
	}
}

func TestAnalyzeCORS_NoIssues(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Access-Control-Allow-Origin":   []string{"https://example.com"},
			"Access-Control-Allow-Headers":  []string{"Content-Type"},
			"Access-Control-Expose-Headers": []string{"X-Trace"},
			"Access-Control-Max-Age":        []string{"600"},
			"Cross-Origin-Resource-Policy":  []string{"same-site"},
			"Vary":                          []string{"Origin"},
		},
	}
	if report := AnalyzeCORS(resp); report != nil {
		t.Fatalf("expected nil report, got %+v", report)
	}
}

func TestAnalyzeCORS_ExtensionIssues(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Access-Control-Allow-Origin":   []string{"https://example.com"},
			"Access-Control-Allow-Headers":  []string{"*"},
			"Access-Control-Expose-Headers": []string{"*,X-Test"},
		},
	}
	report := AnalyzeCORS(resp)
	if report == nil {
		t.Fatal("expected report when insecure extensions are present")
	}
	if !strings.Contains(strings.Join(report.Issues, ","), "allows any header") {
		t.Fatalf("expected issue about wildcard headers, got %v", report.Issues)
	}
	if report.VaryOrigin {
		t.Fatal("expected VaryOrigin to be false when Vary header missing")
	}
	if report.ResourcePolicy != "" {
		t.Fatal("expected ResourcePolicy to be empty when header missing")
	}
	if report.MaxAge != "" {
		t.Fatal("expected MaxAge to be empty when header missing")
	}
}
