package checker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"
)

func TestDiscoverInScopeLinks_Basic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<a href="#/login">Login</a><a href="/#/register">Register</a><a href="/blog">Blog</a>`)
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "login page")
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "register page")
	})
	mux.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "blog page")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	opts := CrawlOptions{MaxDepth: 1, MaxPages: 10, SameHostOnly: true, Timeout: time.Second}
	links, err := DiscoverInScopeLinks(context.Background(), server.URL, opts)
	if err != nil {
		t.Fatalf("DiscoverInScopeLinks returned error: %v", err)
	}

	sort.Strings(links)
	want := []string{server.URL + "/blog", server.URL + "/login", server.URL + "/register"}
	if len(links) != len(want) {
		t.Fatalf("expected %d links, got %d (%v)", len(want), len(links), links)
	}
	for i := range want {
		if links[i] != want[i] {
			t.Errorf("link %d mismatch: want %s, got %s", i, want[i], links[i])
		}
	}
}

func TestDiscoverInScopeLinks_DepthLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<a href="/level1">L1</a>`)
	})
	mux.HandleFunc("/level1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<a href="/level2">L2</a>`)
	})
	mux.HandleFunc("/level2", func(w http.ResponseWriter, r *http.Request) {})
	server := httptest.NewServer(mux)
	defer server.Close()

	ctx := context.Background()
	opts := CrawlOptions{MaxDepth: 1, MaxPages: 10, SameHostOnly: true, Timeout: time.Second}
	links, err := DiscoverInScopeLinks(ctx, server.URL, opts)
	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}
	if len(links) != 1 || links[0] != server.URL+"/level1" {
		t.Fatalf("expected only level1, got %v", links)
	}

	opts.MaxDepth = 2
	links, err = DiscoverInScopeLinks(ctx, server.URL, opts)
	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected two links, got %v", links)
	}
}

func TestDiscoverInScopeLinks_IgnoresExternal(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<a href="https://example.com/phish">external</a><a href="/ok">ok</a>`)
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {})

	server := httptest.NewServer(mux)
	defer server.Close()

	opts := CrawlOptions{MaxDepth: 1, MaxPages: 10, SameHostOnly: true, Timeout: time.Second}
	links, err := DiscoverInScopeLinks(context.Background(), server.URL, opts)
	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}
	if len(links) != 1 || links[0] != server.URL+"/ok" {
		t.Fatalf("expected only /ok, got %v", links)
	}
}

func TestEnsureLeadingSlash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/path", "/path"},
		{"path", "/path"},
		{"/", "/"},
		{"path/to/file", "/path/to/file"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ensureLeadingSlash(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNormalizeSPAPath(t *testing.T) {
	// Test nil URL
	normalizeSPAPath(nil)

	// Test hash paths in URL path (not fragment)
	u, _ := url.Parse("http://example.com")
	u.Path = "/#/login"
	normalizeSPAPath(u)
	if u.Path != "/login" {
		t.Errorf("expected /login, got %s", u.Path)
	}

	// Test normal path (no change)
	u, _ = url.Parse("http://example.com/about")
	normalizeSPAPath(u)
	if u.Path != "/about" {
		t.Errorf("expected /about, got %s", u.Path)
	}
}

func TestCanonicalURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with fragment", "http://example.com/path#section", "http://example.com/path"},
		{"no fragment", "http://example.com/path", "http://example.com/path"},
		{"root", "http://example.com", "http://example.com/"},
		{"with query", "http://example.com/path?q=1#frag", "http://example.com/path?q=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.input)
			result := canonicalURL(u)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}

	// Test nil URL
	if canonicalURL(nil) != "" {
		t.Error("expected empty string for nil URL")
	}
}

func TestIsHTML(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"", true},
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"TEXT/HTML", true},
		{"application/json", false},
		{"image/png", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := isHTML(tt.contentType)
			if result != tt.expected {
				t.Errorf("isHTML(%s) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeAsset(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"", false},
		{"/", false},
		{"/page", false},
		{"/style.css", true},
		{"/script.js", true},
		{"/image.png", true},
		{"/doc.pdf", true},
		{"/video.mp4", true},
		{"/page.html", false}, // HTML is not an asset for crawling
		{"/api/data", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := looksLikeAsset(tt.path)
			if result != tt.expected {
				t.Errorf("looksLikeAsset(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
