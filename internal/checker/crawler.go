package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CrawlOptions configures discovery of in-scope links.
type CrawlOptions struct {
	MaxDepth     int
	MaxPages     int
	SameHostOnly bool
	Timeout      time.Duration
}

const maxCrawlBodyBytes = 512 * 1024

var hrefPattern = regexp.MustCompile(`(?i)href\s*=\s*(?:'([^']*)'|"([^"]*)"|([^\s"'<>]+))`)

var assetExtensions = map[string]struct{}{
	".css":         {},
	".js":          {},
	".json":        {},
	".map":         {},
	".txt":         {},
	".png":         {},
	".jpg":         {},
	".jpeg":        {},
	".gif":         {},
	".svg":         {},
	".ico":         {},
	".webp":        {},
	".webmanifest": {},
	".mp4":         {},
	".mp3":         {},
	".woff":        {},
	".woff2":       {},
	".ttf":         {},
	".eot":         {},
	".pdf":         {},
	".zip":         {},
	".tar":         {},
}

// DiscoverInScopeLinks crawls within the same host starting from startURL and returns
// up to MaxPages canonical URLs discovered within MaxDepth hops.
func DiscoverInScopeLinks(ctx context.Context, startURL string, opts CrawlOptions) ([]string, error) {
	if opts.MaxDepth <= 0 || opts.MaxPages <= 0 {
		return nil, nil
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}

	target := ParseTarget(startURL)
	if target == nil || target.FullURL == "" {
		return nil, fmt.Errorf("invalid start url %q", startURL)
	}

	root, err := url.Parse(target.FullURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: opts.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS12,
			},
		},
	}

	type queueItem struct {
		url   *url.URL
		depth int
	}

	queue := []queueItem{{url: root, depth: 0}}
	seen := map[string]struct{}{canonicalURL(root): {}}
	discovered := make([]string, 0, opts.MaxPages)

	for len(queue) > 0 && len(discovered) < opts.MaxPages {
		if err := ctx.Err(); err != nil {
			return discovered, err
		}

		item := queue[0]
		queue = queue[1:]

		if item.depth >= opts.MaxDepth {
			continue
		}

		body, contentType, err := fetchPage(ctx, client, item.url.String())
		if err != nil || !isHTML(contentType) {
			continue
		}

		links := extractLinks(item.url, body)
		for _, raw := range links {
			u, err := url.Parse(raw)
			if err != nil {
				continue
			}
			if opts.SameHostOnly && !hostsMatch(root, u) {
				continue
			}
			if looksLikeAsset(u.Path) {
				continue
			}
			key := canonicalURL(u)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			discovered = append(discovered, key)
			if len(discovered) >= opts.MaxPages {
				break
			}
			if item.depth+1 < opts.MaxDepth {
				queue = append(queue, queueItem{url: u, depth: item.depth + 1})
			}
		}
	}

	return discovered, nil
}

func fetchPage(ctx context.Context, client *http.Client, target string) ([]byte, string, error) {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
					MinVersion:         tls.VersionTLS12,
				},
			},
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, resp.Header.Get("Content-Type"), fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxCrawlBodyBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, resp.Header.Get("Content-Type"), err
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func extractLinks(base *url.URL, body []byte) []string {
	matches := hrefPattern.FindAllSubmatch(body, -1)
	links := make([]string, 0, len(matches))
	for _, match := range matches {
		var raw string
		for i := 1; i < len(match); i++ {
			if len(match[i]) > 0 {
				raw = string(match[i])
				break
			}
		}
		if raw == "" {
			continue
		}
		if resolved := resolveLink(base, raw); resolved != nil {
			links = append(links, resolved.String())
		}
	}
	return links
}

func resolveLink(base *url.URL, href string) *url.URL {
	href = strings.TrimSpace(href)
	if href == "" {
		return nil
	}
	lower := strings.ToLower(href)
	switch {
	case strings.HasPrefix(lower, "javascript:"),
		strings.HasPrefix(lower, "mailto:"),
		strings.HasPrefix(lower, "tel:"):
		return nil
	}

	if strings.HasPrefix(href, "#/") {
		return buildURLFromPath(base, href[1:])
	}
	if strings.HasPrefix(href, "/#/") {
		return buildURLFromPath(base, href[2:])
	}

	ref, err := url.Parse(href)
	if err != nil {
		return nil
	}
	if ref.Scheme == "" {
		ref = base.ResolveReference(ref)
	}
	if ref == nil {
		return nil
	}
	if ref.Scheme != "http" && ref.Scheme != "https" {
		return nil
	}

	if strings.HasPrefix(ref.Fragment, "/") {
		ref.Path = ensureLeadingSlash(ref.Fragment)
	}
	ref.Fragment = ""
	normalizeSPAPath(ref)
	if ref.Path == "" {
		ref.Path = "/"
	}
	return ref
}

func buildURLFromPath(base *url.URL, path string) *url.URL {
	if base == nil {
		return nil
	}
	return &url.URL{
		Scheme: base.Scheme,
		Host:   base.Host,
		Path:   ensureLeadingSlash(path),
	}
}

func ensureLeadingSlash(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func normalizeSPAPath(u *url.URL) {
	if u == nil {
		return
	}
	switch {
	case strings.HasPrefix(u.Path, "/#/"):
		u.Path = ensureLeadingSlash(strings.TrimPrefix(u.Path, "/#/"))
	case strings.HasPrefix(u.Path, "#/"):
		u.Path = ensureLeadingSlash(strings.TrimPrefix(u.Path, "#/"))
	}
}

func canonicalURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	copy := *u
	copy.Fragment = ""
	if copy.Path == "" {
		copy.Path = "/"
	}
	return copy.String()
}

func hostsMatch(a, b *url.URL) bool {
	return !sameHostEmpty(a) && !sameHostEmpty(b) && strings.EqualFold(a.Hostname(), b.Hostname())
}

func sameHostEmpty(u *url.URL) bool {
	return u == nil || u.Hostname() == ""
}

func isHTML(contentType string) bool {
	if contentType == "" {
		return true
	}
	return strings.Contains(strings.ToLower(contentType), "text/html")
}

func looksLikeAsset(path string) bool {
	if path == "" || path == "/" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}
	_, blocked := assetExtensions[ext]
	return blocked
}
