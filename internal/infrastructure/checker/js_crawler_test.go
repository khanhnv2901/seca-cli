package checker

import (
	"context"
	"testing"
	"time"
)

func TestDiscoverInScopeLinksJS_RealSite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	opts := JSCrawlOptions{
		CrawlOptions: CrawlOptions{
			MaxDepth:     2,
			MaxPages:     20,
			SameHostOnly: true,
			Timeout:      30 * time.Second,
		},
		EnableJavaScript: true,
		WaitTime:         3 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	links, err := DiscoverInScopeLinksJS(ctx, "https://qk3.vti.com.vn", opts)
	if err != nil {
		t.Fatalf("DiscoverInScopeLinksJS failed: %v", err)
	}

	t.Logf("Discovered %d links:", len(links))
	for i, link := range links {
		t.Logf("  %d: %s", i+1, link)
	}

	if len(links) == 0 {
		t.Error("Expected to discover some links, but got none")
	}
}

func TestPageRequiresJavaScript(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantJS   bool
	}{
		{
			name:   "SPA site",
			url:    "https://qk3.vti.com.vn",
			wantJS: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			needsJS, err := pageRequiresJavaScript(ctx, tt.url)
			if err != nil {
				t.Fatalf("pageRequiresJavaScript failed: %v", err)
			}
			if needsJS != tt.wantJS {
				t.Errorf("pageRequiresJavaScript() = %v, want %v", needsJS, tt.wantJS)
			}
		})
	}
}
