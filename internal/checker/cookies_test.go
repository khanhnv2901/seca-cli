package checker

import (
	"net/http"
	"testing"
)

func TestAnalyzeCookies(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Add("Set-Cookie", "session=abc123; Path=/")
	resp.Header.Add("Set-Cookie", "prefs=dark; Path=/; Secure")

	findings := AnalyzeCookies(resp)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	if !findings[0].MissingSecure || !findings[0].MissingHTTPOnly {
		t.Errorf("expected session cookie to miss both flags: %+v", findings[0])
	}

	if findings[1].MissingSecure {
		t.Errorf("expected prefs cookie to include Secure flag")
	}

	if !findings[1].MissingHTTPOnly {
		t.Errorf("expected prefs cookie to miss HttpOnly flag")
	}
}

func TestAnalyzeCookies_NoSetCookie(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	if findings := AnalyzeCookies(resp); len(findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(findings))
	}
}
