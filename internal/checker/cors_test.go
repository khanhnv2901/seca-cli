package checker

import (
	"net/http"
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
			"Access-Control-Allow-Origin": []string{"https://example.com"},
		},
	}
	if report := AnalyzeCORS(resp); report != nil {
		t.Fatalf("expected nil report, got %+v", report)
	}
}
