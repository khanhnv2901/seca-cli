package checker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
