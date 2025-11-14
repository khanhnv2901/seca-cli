package checker

import (
	"net/url"
	"regexp"
	"strings"
)

var scriptSrcPattern = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)

// AnalyzeThirdPartyScripts returns external script URLs (supply-chain visibility).
func AnalyzeThirdPartyScripts(body string, base *url.URL) []string {
	if len(body) == 0 || base == nil {
		return nil
	}

	matches := scriptSrcPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	baseHost := strings.ToLower(base.Hostname())
	seen := make(map[string]struct{})
	scripts := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		src := strings.TrimSpace(match[1])
		if src == "" {
			continue
		}
		resolved, err := resolveScriptURL(src, base)
		if err != nil || resolved == "" {
			continue
		}
		u, err := url.Parse(resolved)
		if err != nil || u.Hostname() == "" {
			continue
		}
		if strings.ToLower(u.Hostname()) == baseHost {
			continue // first-party
		}
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		scripts = append(scripts, resolved)
	}

	return scripts
}

func resolveScriptURL(src string, base *url.URL) (string, error) {
	if strings.HasPrefix(src, "//") {
		return base.Scheme + ":" + src, nil
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src, nil
	}
	if strings.HasPrefix(src, "data:") {
		return "", nil
	}
	resolved, err := base.Parse(src)
	if err != nil {
		return "", err
	}
	return resolved.String(), nil
}
