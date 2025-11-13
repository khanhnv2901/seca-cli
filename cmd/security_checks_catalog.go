package cmd

// SecurityCheckSpec describes a high-level security check and its category.
type SecurityCheckSpec struct {
	Name     string
	Category string
}

// securityCheckCatalog lists every security control documented in
// docs/materials/list-of-security-check.md. Keep this slice in sync with that
// document; security_checks_catalog_test.go validates the contents match.
var securityCheckCatalog = []SecurityCheckSpec{
	{Name: "Frame Security Policy (X-Frame-Options)", Category: "Clickjacking Protection"},
	{Name: "Access-Control-Allow-Credentials header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Access-Control-Allow-Headers header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Access-Control-Allow-Origin header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Access-Control-Expose-Headers header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Access-Control-Max-Age header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Cross-Origin-Embedder-Policy header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Cross-Origin-Opener-Policy header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Cross-Origin-Resource-Policy header", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Cross-Origin Resource Isolation", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Vary: Origin header (CORS caching)", Category: "Cross-Origin Resource Sharing (CORS)"},
	{Name: "Content Security Policy (CSP)", Category: "Content Security Policy (CSP)"},
	{Name: "Content Security Policy (CSP) Bypass", Category: "Content Security Policy (CSP)"},
	{Name: "Set-Cookie headers (Secure/HttpOnly)", Category: "Cookie Security"},
	{Name: "Open Ports", Category: "Network Security"},
	{Name: "Subdomain Takeover", Category: "Network Security"},
	{Name: "Permissions-Policy header", Category: "Miscellaneous Headers"},
	{Name: "Referrer Policy", Category: "Miscellaneous Headers"},
	{Name: "Server information disclosure", Category: "Miscellaneous Headers"},
	{Name: "Content-Type header", Category: "Miscellaneous Headers"},
	{Name: "Deprecated X-XSS-Protection header", Category: "Miscellaneous Headers"},
	{Name: "Vulnerable JS Libraries", Category: "Miscellaneous"},
	{Name: "Anti-CSRF Tokens", Category: "Cross-Site Scripting (XSS) Protection"},
	{Name: "Trusted Types readiness", Category: "Cross-Site Scripting (XSS) Protection"},
	{Name: "X-Content-Type-Options", Category: "Cross-Site Scripting (XSS) Protection"},
	{Name: "Certificate Hostname & Chain", Category: "Transport Layer Security (TLS)"},
	{Name: "Certificate Expiry", Category: "Transport Layer Security (TLS)"},
	{Name: "Cipher Suite", Category: "Transport Layer Security (TLS)"},
	{Name: "Deprecated TLS versions supported", Category: "Transport Layer Security (TLS)"},
	{Name: "HTTPS enabled", Category: "Transport Layer Security (TLS)"},
	{Name: "HSTS enabled", Category: "Transport Layer Security (TLS)"},
	{Name: "Mixed Content", Category: "Transport Layer Security (TLS)"},
	{Name: "OCSP Stapling", Category: "Transport Layer Security (TLS)"},
	{Name: "TLS Version", Category: "Transport Layer Security (TLS)"},
}

func getSecurityCheckCatalog() []SecurityCheckSpec {
	out := make([]SecurityCheckSpec, len(securityCheckCatalog))
	copy(out, securityCheckCatalog)
	return out
}
