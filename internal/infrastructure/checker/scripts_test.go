package checker

import (
	"net/url"
	"testing"
)

func TestAnalyzeThirdPartyScripts(t *testing.T) {
	body := `
	<html>
	  <head>
	    <script src="/static/app.js"></script>
	    <script src="https://cdn.example.com/lib.js"></script>
	    <script SRC='//analytics.example.org/script.js'></script>
	  </head>
	</html>`

	base, _ := url.Parse("https://app.internal.test")
	scripts := AnalyzeThirdPartyScripts(body, base)

	if len(scripts) != 2 {
		t.Fatalf("expected 2 third-party scripts, got %d", len(scripts))
	}

	expectedHosts := map[string]bool{
		"cdn.example.com":       false,
		"analytics.example.org": false,
	}

	for _, s := range scripts {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatalf("invalid URL in results: %s", s)
		}
		if _, ok := expectedHosts[u.Hostname()]; !ok {
			t.Fatalf("unexpected host %s", u.Hostname())
		}
		expectedHosts[u.Hostname()] = true
	}

	for host, seen := range expectedHosts {
		if !seen {
			t.Errorf("expected script host %s not returned", host)
		}
	}
}

func TestAnalyzeThirdPartyScripts_None(t *testing.T) {
	base, _ := url.Parse("https://example.com")
	if res := AnalyzeThirdPartyScripts(`<script src="/app.js"></script>`, base); len(res) != 0 {
		t.Fatalf("expected no third-party scripts, got %v", res)
	}
}
