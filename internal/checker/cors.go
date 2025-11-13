package checker

import (
	"net/http"
	"strings"
)

// AnalyzeCORS inspects CORS headers for insecure defaults (OWASP A5:2021).
func AnalyzeCORS(resp *http.Response) *CORSReport {
	if resp == nil {
		return nil
	}
	headers := resp.Header
	report := &CORSReport{
		AllowOrigin:      headers.Get("Access-Control-Allow-Origin"),
		AllowMethods:     headers.Get("Access-Control-Allow-Methods"),
		AllowHeaders:     headers.Get("Access-Control-Allow-Headers"),
		ExposeHeaders:    headers.Get("Access-Control-Expose-Headers"),
		MaxAge:           headers.Get("Access-Control-Max-Age"),
		ResourcePolicy:   headers.Get("Cross-Origin-Resource-Policy"),
		AllowCredentials: headers.Get("Access-Control-Allow-Credentials") == "true",
		VaryOrigin:       varyIncludesOrigin(headers.Values("Vary")),
	}

	if report.AllowOrigin == "" {
		report.MissingAllowOrigin = true
		report.Issues = append(report.Issues, "Access-Control-Allow-Origin header missing")
	} else if report.AllowOrigin == "*" {
		report.AllowsAnyOrigin = true
		report.Issues = append(report.Issues, "CORS allows any origin (*)")
		if report.AllowCredentials {
			report.Issues = append(report.Issues, "Credentials allowed with wildcard origin (disallowed by browsers)")
		}
	}

	if report.AllowHeaders != "" && strings.Contains(report.AllowHeaders, "*") {
		report.Issues = append(report.Issues, "Access-Control-Allow-Headers allows any header (*)")
	}

	if report.ExposeHeaders != "" && strings.Contains(report.ExposeHeaders, "*") {
		report.Issues = append(report.Issues, "Access-Control-Expose-Headers exposes all headers (*)")
	}

	if report.MaxAge == "" {
		report.Issues = append(report.Issues, "Access-Control-Max-Age header missing (preflight responses may not be cached)")
	}

	if report.ResourcePolicy == "" {
		report.Issues = append(report.Issues, "Cross-Origin-Resource-Policy header missing")
	}

	if !report.VaryOrigin && report.AllowOrigin != "" && !report.AllowsAnyOrigin {
		report.Issues = append(report.Issues, "Vary: Origin header missing (responses may be cached incorrectly)")
	}

	if len(report.Issues) == 0 && !report.MissingAllowOrigin {
		// No risks detected; return nil to avoid cluttering results.
		return nil
	}
	return report
}

func varyIncludesOrigin(values []string) bool {
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), "origin") {
				return true
			}
		}
	}
	return false
}
