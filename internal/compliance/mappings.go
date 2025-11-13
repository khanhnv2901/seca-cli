package compliance

// GetComplianceMappings returns the mapping of security checks to compliance requirements
func GetComplianceMappings() map[string]ComplianceMapping {
	return map[string]ComplianceMapping{
		// Transport Layer Security (TLS) Checks
		"HTTPS enabled": {
			CheckName: "HTTPS enabled",
			Frameworks: map[string][]string{
				"iso27001":    {"A.8.24", "A.8.9"},
				"iso27701":    {"7.4.8"},
				"jisq27001":   {"A.8.24", "A.10.1.1"},
				"pdpa":        {"Protection Obligation 24"},
				"mtcs":        {"CC-02", "IVS-05"},
				"kisms":       {"2.8.1", "2.8.2"},
				"ismsp":       {"2.8.1", "3.1.2"},
				"fisc":        {"Network Security 3-1"},
				"privacymark": {"3.4.2"},
			},
			Priority: map[string]string{
				"iso27001": "Critical", "iso27701": "Critical", "jisq27001": "Critical",
				"pdpa": "Critical", "mtcs": "Critical", "kisms": "Critical",
				"ismsp": "Critical", "fisc": "Critical", "privacymark": "Critical",
			},
		},
		"TLS Version": {
			CheckName: "TLS Version",
			Frameworks: map[string][]string{
				"iso27001":    {"A.8.24"},
				"jisq27001":   {"A.8.24"},
				"pdpa":        {"Protection Obligation 24"},
				"mtcs":        {"CC-02"},
				"kisms":       {"2.8.2"},
				"ismsp":       {"2.8.2"},
				"fisc":        {"Network Security 3-2"},
				"privacymark": {"3.4.2"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "pdpa": "High",
				"mtcs": "High", "kisms": "High", "ismsp": "High",
				"fisc": "High", "privacymark": "High",
			},
		},
		"Deprecated TLS versions supported": {
			CheckName: "Deprecated TLS versions supported",
			Frameworks: map[string][]string{
				"iso27001":    {"A.8.24"},
				"jisq27001":   {"A.8.24"},
				"pdpa":        {"Protection Obligation 24"},
				"mtcs":        {"CC-02"},
				"kisms":       {"2.8.2"},
				"ismsp":       {"2.8.2"},
				"fisc":        {"Network Security 3-2"},
			},
			Priority: map[string]string{
				"iso27001": "Critical", "jisq27001": "Critical", "pdpa": "Critical",
				"mtcs": "Critical", "kisms": "Critical", "ismsp": "Critical",
				"fisc": "Critical",
			},
		},
		"Cipher Suite": {
			CheckName: "Cipher Suite",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.24"},
				"jisq27001": {"A.8.24"},
				"mtcs":      {"CC-02"},
				"kisms":     {"2.8.2"},
				"fisc":      {"Network Security 3-2"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "mtcs": "High",
				"kisms": "High", "fisc": "High",
			},
		},
		"Certificate Hostname & Chain": {
			CheckName: "Certificate Hostname & Chain",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.24"},
				"jisq27001": {"A.8.24"},
				"mtcs":      {"IVS-05"},
				"kisms":     {"2.8.2"},
				"fisc":      {"Network Security 3-3"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "mtcs": "High",
				"kisms": "High", "fisc": "High",
			},
		},
		"Certificate Expiry": {
			CheckName: "Certificate Expiry",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.24"},
				"jisq27001": {"A.8.24"},
				"mtcs":      {"IVS-05"},
				"kisms":     {"2.8.2"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium", "mtcs": "Medium",
				"kisms": "Medium",
			},
		},
		"HSTS enabled": {
			CheckName: "HSTS enabled",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.24"},
				"jisq27001": {"A.8.24"},
				"mtcs":      {"CC-02"},
				"kisms":     {"2.8.1"},
				"fisc":      {"Network Security 3-1"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "mtcs": "High",
				"kisms": "High", "fisc": "High",
			},
		},
		"Mixed Content": {
			CheckName: "Mixed Content",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.24"},
				"jisq27001": {"A.8.24"},
				"kisms":     {"2.8.1"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "kisms": "High",
			},
		},
		"OCSP Stapling": {
			CheckName: "OCSP Stapling",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.24"},
				"jisq27001": {"A.8.24"},
			},
			Priority: map[string]string{
				"iso27001": "Low", "jisq27001": "Low",
			},
		},

		// Content Security Policy
		"Content Security Policy (CSP)": {
			CheckName: "Content Security Policy (CSP)",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16", "A.8.23"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.3"},
				"ismsp":     {"2.7.3"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "kisms": "High",
				"ismsp": "High",
			},
		},
		"Content Security Policy (CSP) Bypass": {
			CheckName: "Content Security Policy (CSP) Bypass",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16", "A.8.23"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.3"},
				"ismsp":     {"2.7.3"},
			},
			Priority: map[string]string{
				"iso27001": "Critical", "jisq27001": "Critical", "kisms": "Critical",
				"ismsp": "Critical",
			},
		},

		// Cross-Site Scripting (XSS) Protection
		"X-Content-Type-Options": {
			CheckName: "X-Content-Type-Options",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.3"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium", "kisms": "Medium",
			},
		},
		"Anti-CSRF Tokens": {
			CheckName: "Anti-CSRF Tokens",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.3"},
				"ismsp":     {"2.7.3"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "kisms": "High",
				"ismsp": "High",
			},
		},
		"Trusted Types readiness": {
			CheckName: "Trusted Types readiness",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.3"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium", "kisms": "Medium",
			},
		},
		"Deprecated X-XSS-Protection header": {
			CheckName: "Deprecated X-XSS-Protection header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Low", "jisq27001": "Low",
			},
		},

		// Clickjacking Protection
		"Frame Security Policy (X-Frame-Options)": {
			CheckName: "Frame Security Policy (X-Frame-Options)",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.3"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium", "kisms": "Medium",
			},
		},

		// CORS
		"Access-Control-Allow-Origin header": {
			CheckName: "Access-Control-Allow-Origin header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16", "A.8.20"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.1"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "kisms": "High",
			},
		},
		"Access-Control-Allow-Credentials header": {
			CheckName: "Access-Control-Allow-Credentials header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16", "A.8.20"},
				"jisq27001": {"A.8.16"},
				"kisms":     {"2.7.1"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "kisms": "High",
			},
		},
		"Access-Control-Allow-Headers header": {
			CheckName: "Access-Control-Allow-Headers header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium",
			},
		},
		"Access-Control-Expose-Headers header": {
			CheckName: "Access-Control-Expose-Headers header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium",
			},
		},
		"Access-Control-Max-Age header": {
			CheckName: "Access-Control-Max-Age header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Low", "jisq27001": "Low",
			},
		},
		"Cross-Origin-Embedder-Policy header": {
			CheckName: "Cross-Origin-Embedder-Policy header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium",
			},
		},
		"Cross-Origin-Opener-Policy header": {
			CheckName: "Cross-Origin-Opener-Policy header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium",
			},
		},
		"Cross-Origin-Resource-Policy header": {
			CheckName: "Cross-Origin-Resource-Policy header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium",
			},
		},
		"Cross-Origin Resource Isolation": {
			CheckName: "Cross-Origin Resource Isolation",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "jisq27001": "Medium",
			},
		},
		"Vary: Origin header (CORS caching)": {
			CheckName: "Vary: Origin header (CORS caching)",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Low", "jisq27001": "Low",
			},
		},

		// Cookie Security
		"Set-Cookie headers (Secure/HttpOnly)": {
			CheckName: "Set-Cookie headers (Secure/HttpOnly)",
			Frameworks: map[string][]string{
				"iso27001":    {"A.8.16"},
				"iso27701":    {"7.2.2"},
				"jisq27001":   {"A.8.16"},
				"pdpa":        {"Protection Obligation 24"},
				"kisms":       {"2.7.2"},
				"ismsp":       {"2.7.2", "3.1.3"},
				"privacymark": {"3.4.2"},
				"pims":        {"3.1.3"},
			},
			Priority: map[string]string{
				"iso27001": "High", "iso27701": "High", "jisq27001": "High",
				"pdpa": "High", "kisms": "High", "ismsp": "High",
				"privacymark": "High", "pims": "High",
			},
		},

		// Network Security
		"Subdomain Takeover": {
			CheckName: "Subdomain Takeover",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.20", "A.8.21"},
				"jisq27001": {"A.8.20"},
				"mtcs":      {"IVS-06"},
				"kisms":     {"2.6.1"},
				"fisc":      {"Network Security 2-1"},
			},
			Priority: map[string]string{
				"iso27001": "Critical", "jisq27001": "Critical", "mtcs": "Critical",
				"kisms": "Critical", "fisc": "Critical",
			},
		},
		"Open Ports": {
			CheckName: "Open Ports",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.20"},
				"jisq27001": {"A.8.20"},
				"mtcs":      {"IVS-06"},
				"kisms":     {"2.6.1", "2.6.2"},
				"fisc":      {"Network Security 2-2"},
			},
			Priority: map[string]string{
				"iso27001": "High", "jisq27001": "High", "mtcs": "High",
				"kisms": "High", "fisc": "High",
			},
		},

		// Miscellaneous Headers
		"Referrer Policy": {
			CheckName: "Referrer Policy",
			Frameworks: map[string][]string{
				"iso27001":    {"A.8.11"},
				"iso27701":    {"7.2.2"},
				"jisq27001":   {"A.8.11"},
				"pdpa":        {"Protection Obligation 21"},
				"ismsp":       {"3.1.4"},
				"privacymark": {"3.4.3"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "iso27701": "Medium", "jisq27001": "Medium",
				"pdpa": "Medium", "ismsp": "Medium", "privacymark": "Medium",
			},
		},
		"Server information disclosure": {
			CheckName: "Server information disclosure",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.9"},
				"jisq27001": {"A.8.9"},
				"kisms":     {"2.7.4"},
			},
			Priority: map[string]string{
				"iso27001": "Low", "jisq27001": "Low", "kisms": "Low",
			},
		},
		"Content-Type header": {
			CheckName: "Content-Type header",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.16"},
				"jisq27001": {"A.8.16"},
			},
			Priority: map[string]string{
				"iso27001": "Low", "jisq27001": "Low",
			},
		},
		"Permissions-Policy header": {
			CheckName: "Permissions-Policy header",
			Frameworks: map[string][]string{
				"iso27001":    {"A.8.16"},
				"iso27701":    {"7.2.2"},
				"jisq27001":   {"A.8.16"},
				"pdpa":        {"Protection Obligation 21"},
				"privacymark": {"3.4.3"},
			},
			Priority: map[string]string{
				"iso27001": "Medium", "iso27701": "Medium", "jisq27001": "Medium",
				"pdpa": "Medium", "privacymark": "Medium",
			},
		},

		// Miscellaneous
		"Vulnerable JS Libraries": {
			CheckName: "Vulnerable JS Libraries",
			Frameworks: map[string][]string{
				"iso27001":  {"A.8.8", "A.8.19"},
				"jisq27001": {"A.8.8"},
				"mtcs":      {"SEF-04"},
				"kisms":     {"2.5.3"},
				"fisc":      {"System Development 4-3"},
			},
			Priority: map[string]string{
				"iso27001": "Critical", "jisq27001": "Critical", "mtcs": "Critical",
				"kisms": "Critical", "fisc": "Critical",
			},
		},
	}
}

// GetMappingForCheck returns compliance mapping for a specific security check
func GetMappingForCheck(checkName string) *ComplianceMapping {
	mappings := GetComplianceMappings()
	if mapping, ok := mappings[checkName]; ok {
		return &mapping
	}
	return nil
}

// GetChecksForFramework returns all security checks relevant to a framework
func GetChecksForFramework(frameworkID string) []string {
	var checks []string
	mappings := GetComplianceMappings()

	for checkName, mapping := range mappings {
		if _, ok := mapping.Frameworks[frameworkID]; ok {
			checks = append(checks, checkName)
		}
	}

	return checks
}

// GetRequirementsForFramework returns all requirements for a specific framework
func GetRequirementsForFramework(frameworkID string) map[string][]string {
	requirements := make(map[string][]string)
	mappings := GetComplianceMappings()

	for checkName, mapping := range mappings {
		if reqs, ok := mapping.Frameworks[frameworkID]; ok {
			requirements[checkName] = reqs
		}
	}

	return requirements
}
