package checker

import "net/http"

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
		AllowCredentials: headers.Get("Access-Control-Allow-Credentials") == "true",
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

	if len(report.Issues) == 0 && !report.MissingAllowOrigin {
		// No risks detected; return nil to avoid cluttering results.
		return nil
	}
	return report
}
