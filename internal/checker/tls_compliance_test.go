package checker

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"strings"
	"testing"
	"time"
)

func TestTLSVersionString(t *testing.T) {
	tests := []struct {
		version  uint16
		expected string
	}{
		{tls.VersionSSL30, "SSL 3.0"},
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
	}

	for _, tt := range tests {
		result := tlsVersionString(tt.version)
		if result != tt.expected {
			t.Errorf("tlsVersionString(0x%04x) = %s, want %s", tt.version, result, tt.expected)
		}
	}
}

func TestAnalyzeTLSCompliance_NilConnectionState(t *testing.T) {
	result := AnalyzeTLSCompliance(nil)
	if result != nil {
		t.Error("Expected nil result for nil connection state")
	}
}

func TestAnalyzeTLSCompliance_TLS12_StrongCipher(t *testing.T) {
	connState := &tls.ConnectionState{
		Version:     tls.VersionTLS12,
		CipherSuite: tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}

	result := AnalyzeTLSCompliance(connState)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.Compliant {
		t.Error("Expected compliant result for TLS 1.2 with strong cipher")
	}

	if result.TLSVersion != "TLS 1.2" {
		t.Errorf("Expected TLS 1.2, got %s", result.TLSVersion)
	}

	if !result.Standards.OWASPASVS9.Compliant {
		t.Error("Expected OWASP ASVS compliance")
	}

	if !result.Standards.PCIDSS41.Compliant {
		t.Error("Expected PCI DSS compliance")
	}

	if len(result.Issues) > 0 {
		t.Errorf("Expected no issues, got %d", len(result.Issues))
	}
}

func TestAnalyzeTLSCompliance_TLS13(t *testing.T) {
	connState := &tls.ConnectionState{
		Version:     tls.VersionTLS13,
		CipherSuite: tls.TLS_AES_256_GCM_SHA384,
	}

	result := AnalyzeTLSCompliance(connState)

	if !result.Compliant {
		t.Error("Expected compliant result for TLS 1.3")
	}

	if result.TLSVersion != "TLS 1.3" {
		t.Errorf("Expected TLS 1.3, got %s", result.TLSVersion)
	}

	// TLS 1.3 should pass both standards
	if !result.Standards.OWASPASVS9.Compliant || !result.Standards.PCIDSS41.Compliant {
		t.Error("TLS 1.3 should be compliant with both standards")
	}
}

func TestAnalyzeTLSCompliance_TLS10_NonCompliant(t *testing.T) {
	connState := &tls.ConnectionState{
		Version:     tls.VersionTLS10,
		CipherSuite: tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}

	result := AnalyzeTLSCompliance(connState)

	if result.Compliant {
		t.Error("Expected non-compliant result for TLS 1.0")
	}

	if result.TLSVersion != "TLS 1.0" {
		t.Errorf("Expected TLS 1.0, got %s", result.TLSVersion)
	}

	if result.Standards.OWASPASVS9.Compliant {
		t.Error("TLS 1.0 should not be ASVS compliant")
	}

	if result.Standards.PCIDSS41.Compliant {
		t.Error("TLS 1.0 should not be PCI DSS compliant")
	}

	if len(result.Issues) == 0 {
		t.Error("Expected issues for TLS 1.0")
	}

	// Check for critical severity
	hasCritical := false
	for _, issue := range result.Issues {
		if issue.Severity == "critical" {
			hasCritical = true
			break
		}
	}
	if !hasCritical {
		t.Error("Expected at least one critical issue for TLS 1.0")
	}
}

func TestAnalyzeTLSCompliance_WeakCipher(t *testing.T) {
	connState := &tls.ConnectionState{
		Version:     tls.VersionTLS12,
		CipherSuite: tls.TLS_RSA_WITH_RC4_128_SHA,
	}

	result := AnalyzeTLSCompliance(connState)

	if result.Compliant {
		t.Error("Expected non-compliant result for weak cipher")
	}

	if len(result.Issues) == 0 {
		t.Error("Expected issues for weak cipher suite")
	}

	// Check for cipher suite issue
	foundCipherIssue := false
	for _, issue := range result.Issues {
		if issue.Requirement == "9.1.2" {
			foundCipherIssue = true
			break
		}
	}
	if !foundCipherIssue {
		t.Error("Expected cipher suite compliance issue")
	}
}

func TestAnalyzeTLSCompliance_WithValidCertificate(t *testing.T) {
	// Create a mock certificate
	now := time.Now()
	cert := &x509.Certificate{
		NotBefore: now.Add(-30 * 24 * time.Hour),
		NotAfter:  now.Add(365 * 24 * time.Hour),
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		Issuer: pkix.Name{
			CommonName: "Example CA",
		},
		DNSNames:            []string{"example.com", "www.example.com"},
		SignatureAlgorithm:  x509.SHA256WithRSA,
		PublicKeyAlgorithm:  x509.RSA,
	}

	connState := &tls.ConnectionState{
		Version:           tls.VersionTLS12,
		CipherSuite:       tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		PeerCertificates: []*x509.Certificate{cert},
	}

	result := AnalyzeTLSCompliance(connState)

	if result.CertificateInfo == nil {
		t.Fatal("Expected certificate info")
	}

	if result.CertificateInfo.DaysUntilExpiry < 360 || result.CertificateInfo.DaysUntilExpiry > 370 {
		t.Errorf("Expected ~365 days until expiry, got %d", result.CertificateInfo.DaysUntilExpiry)
	}

	if result.CertificateInfo.ValidChain != true {
		t.Error("Expected valid chain")
	}

	if result.CertificateInfo.SignatureAlg != "SHA256-RSA" {
		t.Errorf("Expected SHA256-RSA, got %s", result.CertificateInfo.SignatureAlg)
	}
}

func TestAnalyzeTLSCompliance_ExpiredCertificate(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		NotBefore: now.Add(-400 * 24 * time.Hour),
		NotAfter:  now.Add(-10 * 24 * time.Hour), // Expired 10 days ago
		Subject: pkix.Name{
			CommonName: "expired.example.com",
		},
		Issuer: pkix.Name{
			CommonName: "Example CA",
		},
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	connState := &tls.ConnectionState{
		Version:           tls.VersionTLS12,
		CipherSuite:       tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		PeerCertificates: []*x509.Certificate{cert},
	}

	result := AnalyzeTLSCompliance(connState)

	if result.Compliant {
		t.Error("Expected non-compliant result for expired certificate")
	}

	if result.CertificateInfo.DaysUntilExpiry >= 0 {
		t.Errorf("Expected negative days for expired cert, got %d", result.CertificateInfo.DaysUntilExpiry)
	}

	// Should have certificate expiry issue
	foundExpiryIssue := false
	for _, issue := range result.Issues {
		if issue.Severity == "critical" && issue.Description == "Certificate has expired" {
			foundExpiryIssue = true
			break
		}
	}
	if !foundExpiryIssue {
		t.Error("Expected critical certificate expiry issue")
	}
}

func TestAnalyzeTLSCompliance_SelfSignedCertificate(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		NotBefore: now.Add(-30 * 24 * time.Hour),
		NotAfter:  now.Add(365 * 24 * time.Hour),
		Subject: pkix.Name{
			CommonName: "self-signed.example.com",
		},
		Issuer: pkix.Name{
			CommonName: "self-signed.example.com", // Same as subject = self-signed
		},
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	connState := &tls.ConnectionState{
		Version:           tls.VersionTLS12,
		CipherSuite:       tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		PeerCertificates: []*x509.Certificate{cert},
	}

	result := AnalyzeTLSCompliance(connState)

	if result.CertificateInfo == nil {
		t.Fatal("Expected certificate info")
	}

	if !result.CertificateInfo.SelfSigned {
		t.Error("Expected self-signed flag to be true")
	}

	// Should have recommendation about self-signed cert
	foundRecommendation := false
	for _, rec := range result.Recommendations {
		if containsString(rec, "self-signed") || containsString(rec, "Self-signed") {
			foundRecommendation = true
			break
		}
	}
	if !foundRecommendation {
		t.Error("Expected recommendation about self-signed certificate")
	}
}

func TestCheckTLSVersion_AllVersions(t *testing.T) {
	tests := []struct {
		name            string
		version         uint16
		shouldBeCompliant bool
	}{
		{"SSL 3.0", tls.VersionSSL30, false},
		{"TLS 1.0", tls.VersionTLS10, false},
		{"TLS 1.1", tls.VersionTLS11, false},
		{"TLS 1.2", tls.VersionTLS12, true},
		{"TLS 1.3", tls.VersionTLS13, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connState := &tls.ConnectionState{
				Version:     tt.version,
				CipherSuite: tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			}

			result := &TLSComplianceResult{
				Compliant: true,
				Standards: ComplianceStandards{
					OWASPASVS9: ComplianceStatus{Compliant: true, Level: "L1"},
					PCIDSS41:   ComplianceStatus{Compliant: true},
				},
				Issues:          []ComplianceIssue{},
				Recommendations: []string{},
			}

			result.TLSVersion = tlsVersionString(connState.Version)
			checkTLSVersion(connState, result)

			if result.Standards.OWASPASVS9.Compliant != tt.shouldBeCompliant {
				t.Errorf("%s: ASVS compliance = %v, want %v", tt.name, result.Standards.OWASPASVS9.Compliant, tt.shouldBeCompliant)
			}

			if result.Standards.PCIDSS41.Compliant != tt.shouldBeCompliant {
				t.Errorf("%s: PCI DSS compliance = %v, want %v", tt.name, result.Standards.PCIDSS41.Compliant, tt.shouldBeCompliant)
			}
		})
	}
}

func TestComplianceStandards_Structure(t *testing.T) {
	connState := &tls.ConnectionState{
		Version:     tls.VersionTLS12,
		CipherSuite: tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	}

	result := AnalyzeTLSCompliance(connState)

	// Check OWASP ASVS structure
	if result.Standards.OWASPASVS9.Level != "L1" {
		t.Errorf("Expected ASVS Level L1, got %s", result.Standards.OWASPASVS9.Level)
	}

	if result.Standards.OWASPASVS9.Passed == nil {
		t.Error("Expected Passed array to be initialized")
	}

	if result.Standards.OWASPASVS9.Failed == nil {
		t.Error("Expected Failed array to be initialized")
	}

	// Check PCI DSS structure
	if result.Standards.PCIDSS41.Passed == nil {
		t.Error("Expected Passed array to be initialized")
	}

	if result.Standards.PCIDSS41.Failed == nil {
		t.Error("Expected Failed array to be initialized")
	}
}

func TestComplianceIssue_Fields(t *testing.T) {
	connState := &tls.ConnectionState{
		Version:     tls.VersionTLS10,
		CipherSuite: tls.TLS_RSA_WITH_RC4_128_SHA,
	}

	result := AnalyzeTLSCompliance(connState)

	if len(result.Issues) == 0 {
		t.Fatal("Expected issues for TLS 1.0 with weak cipher")
	}

	for _, issue := range result.Issues {
		if issue.Standard == "" {
			t.Error("Issue missing Standard field")
		}
		if issue.Requirement == "" {
			t.Error("Issue missing Requirement field")
		}
		if issue.Severity == "" {
			t.Error("Issue missing Severity field")
		}
		if issue.Description == "" {
			t.Error("Issue missing Description field")
		}
		if issue.Remediation == "" {
			t.Error("Issue missing Remediation field")
		}
	}
}

func TestCipherSuiteName_Coverage(t *testing.T) {
	// Test that all known cipher suites have names
	suites := []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_RC4_128_SHA,
	}

	for _, suite := range suites {
		name := cipherSuiteString(suite)
		if name == "" || strings.HasPrefix(name, "Unknown") {
			t.Errorf("Cipher suite 0x%04x has no proper name: %s", suite, name)
		}
	}
}

// Helper function
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
