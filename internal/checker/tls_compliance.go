package checker

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"
)

// versionSSL30 represents the legacy SSL 3.0 protocol version (0x0300).
// Defined locally so we can detect/report SSL 3.0 without referencing the
// deprecated tls.VersionSSL30 symbol.
const versionSSL30 uint16 = 0x0300

// Weak cipher suites that should not be used (PCI DSS 4.1)
var weakCipherSuites = map[uint16]string{
	tls.TLS_RSA_WITH_RC4_128_SHA:                "TLS_RSA_WITH_RC4_128_SHA",
	tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:           "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_RSA_WITH_AES_128_CBC_SHA:            "TLS_RSA_WITH_AES_128_CBC_SHA (weak)",
	tls.TLS_RSA_WITH_AES_256_CBC_SHA:            "TLS_RSA_WITH_AES_256_CBC_SHA (weak)",
	tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:        "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:          "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:     "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256 (CBC mode)",
}

// Approved strong cipher suites for TLS 1.2/1.3 (OWASP ASVS §9, PCI DSS 4.1)
var strongCipherSuites = map[uint16]string{
	// TLS 1.3 cipher suites (all considered strong)
	tls.TLS_AES_128_GCM_SHA256:       "TLS_AES_128_GCM_SHA256",
	tls.TLS_AES_256_GCM_SHA384:       "TLS_AES_256_GCM_SHA384",
	tls.TLS_CHACHA20_POLY1305_SHA256: "TLS_CHACHA20_POLY1305_SHA256",
	// TLS 1.2 strong cipher suites
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:    "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305:  "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
}

// AnalyzeTLSCompliance analyzes TLS connection for OWASP ASVS §9 and PCI DSS 4.1 compliance
func AnalyzeTLSCompliance(connState *tls.ConnectionState) *TLSComplianceResult {
	if connState == nil {
		return nil
	}

	result := &TLSComplianceResult{
		Compliant:       true,
		TLSVersion:      tlsVersionString(connState.Version),
		CipherSuite:     cipherSuiteString(connState.CipherSuite),
		Protocol:        connState.NegotiatedProtocol,
		Issues:          []ComplianceIssue{},
		Recommendations: []string{},
		Standards: ComplianceStandards{
			OWASPASVS9: ComplianceStatus{
				Compliant: true,
				Level:     "L1",
				Passed:    []string{},
				Failed:    []string{},
			},
			PCIDSS41: ComplianceStatus{
				Compliant: true,
				Passed:    []string{},
				Failed:    []string{},
			},
		},
	}

	// Check TLS version (ASVS 9.1.3, PCI DSS Requirement 4)
	checkTLSVersion(connState, result)

	// Check cipher suite strength (ASVS 9.1.2, PCI DSS 4.1)
	checkCipherSuite(connState, result)

	// Analyze certificate (ASVS 9.2.1, PCI DSS 4.1)
	if len(connState.PeerCertificates) > 0 {
		result.CertificateInfo = analyzeCertificate(connState.PeerCertificates[0])
		checkCertificateCompliance(result.CertificateInfo, result)
	}

	// Overall compliance determination
	result.Compliant = result.Standards.OWASPASVS9.Compliant && result.Standards.PCIDSS41.Compliant

	return result
}

// checkTLSVersion validates TLS protocol version
func checkTLSVersion(connState *tls.ConnectionState, result *TLSComplianceResult) {
	version := connState.Version

	// OWASP ASVS 9.1.3: Only TLS 1.2 and TLS 1.3 allowed (Level 1)
	// PCI DSS 4.1: Only TLS 1.2+ with strong cryptography
	if version < tls.VersionTLS12 {
		issue := ComplianceIssue{
			Standard:    "OWASP ASVS 9.1.3 / PCI DSS 4.1",
			Requirement: "9.1.3",
			Severity:    "critical",
			Description: fmt.Sprintf("Insecure TLS version: %s. Only TLS 1.2 and TLS 1.3 are allowed.", result.TLSVersion),
			Remediation: "Upgrade to TLS 1.2 or TLS 1.3. Disable SSL 2.0, SSL 3.0, TLS 1.0, and TLS 1.1.",
		}
		result.Issues = append(result.Issues, issue)
		result.Standards.OWASPASVS9.Failed = append(result.Standards.OWASPASVS9.Failed, "9.1.3")
		result.Standards.OWASPASVS9.Compliant = false
		result.Standards.PCIDSS41.Failed = append(result.Standards.PCIDSS41.Failed, "4.1")
		result.Standards.PCIDSS41.Compliant = false
		result.Recommendations = append(result.Recommendations,
			"CRITICAL: Upgrade to TLS 1.2 or TLS 1.3 immediately")
	} else if version == tls.VersionTLS12 {
		result.Standards.OWASPASVS9.Passed = append(result.Standards.OWASPASVS9.Passed, "9.1.3")
		result.Standards.PCIDSS41.Passed = append(result.Standards.PCIDSS41.Passed, "4.1-TLS-Version")
		result.Recommendations = append(result.Recommendations,
			"Consider upgrading to TLS 1.3 for improved security and performance")
	} else if version == tls.VersionTLS13 {
		result.Standards.OWASPASVS9.Passed = append(result.Standards.OWASPASVS9.Passed, "9.1.3")
		result.Standards.PCIDSS41.Passed = append(result.Standards.PCIDSS41.Passed, "4.1-TLS-Version")
	}
}

// checkCipherSuite validates cipher suite strength
func checkCipherSuite(connState *tls.ConnectionState, result *TLSComplianceResult) {
	cipherSuite := connState.CipherSuite

	// Check if cipher suite is explicitly weak
	if weakName, isWeak := weakCipherSuites[cipherSuite]; isWeak {
		issue := ComplianceIssue{
			Standard:    "OWASP ASVS 9.1.2 / PCI DSS 4.1",
			Requirement: "9.1.2",
			Severity:    "high",
			Description: fmt.Sprintf("Weak cipher suite detected: %s", weakName),
			Remediation: "Use strong cipher suites with AEAD (Authenticated Encryption with Associated Data) like AES-GCM or ChaCha20-Poly1305.",
		}
		result.Issues = append(result.Issues, issue)
		result.Standards.OWASPASVS9.Failed = append(result.Standards.OWASPASVS9.Failed, "9.1.2")
		result.Standards.OWASPASVS9.Compliant = false
		result.Standards.PCIDSS41.Failed = append(result.Standards.PCIDSS41.Failed, "4.1-Cipher")
		result.Standards.PCIDSS41.Compliant = false
	} else if _, isStrong := strongCipherSuites[cipherSuite]; isStrong {
		// Strong cipher suite
		result.Standards.OWASPASVS9.Passed = append(result.Standards.OWASPASVS9.Passed, "9.1.2")
		result.Standards.PCIDSS41.Passed = append(result.Standards.PCIDSS41.Passed, "4.1-Cipher")
	} else {
		// Unknown cipher suite - warn but don't fail
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Cipher suite %s is not in the recommended strong cipher list. Verify it meets security requirements.", result.CipherSuite))
	}

	// Check for forward secrecy (ECDHE or DHE key exchange)
	cipherName := result.CipherSuite
	if !strings.Contains(cipherName, "ECDHE") && !strings.Contains(cipherName, "DHE") && connState.Version < tls.VersionTLS13 {
		issue := ComplianceIssue{
			Standard:    "OWASP ASVS 9.1.2",
			Requirement: "9.1.2",
			Severity:    "medium",
			Description: "Cipher suite does not provide Perfect Forward Secrecy (PFS)",
			Remediation: "Use cipher suites with ECDHE or DHE key exchange for Perfect Forward Secrecy.",
		}
		result.Issues = append(result.Issues, issue)
		result.Recommendations = append(result.Recommendations,
			"Prefer cipher suites with Perfect Forward Secrecy (ECDHE/DHE)")
	}
}

// analyzeCertificate extracts certificate information
func analyzeCertificate(cert *x509.Certificate) *CertificateInfo {
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

	info := &CertificateInfo{
		Subject:         cert.Subject.String(),
		Issuer:          cert.Issuer.String(),
		NotBefore:       cert.NotBefore.Format(time.RFC3339),
		NotAfter:        cert.NotAfter.Format(time.RFC3339),
		DNSNames:        cert.DNSNames,
		SelfSigned:      cert.Subject.String() == cert.Issuer.String(),
		ValidChain:      false, // Will be determined by TLS handshake success
		DaysUntilExpiry: daysUntilExpiry,
		SignatureAlg:    cert.SignatureAlgorithm.String(),
		PublicKeyAlg:    cert.PublicKeyAlgorithm.String(),
	}

	// Extract key size based on public key type
	switch pubKey := cert.PublicKey.(type) {
	case interface{ Size() int }:
		info.KeySize = pubKey.Size() * 8 // Convert bytes to bits
	}

	return info
}

// checkCertificateCompliance validates certificate against ASVS and PCI DSS requirements
func checkCertificateCompliance(certInfo *CertificateInfo, result *TLSComplianceResult) {
	// ASVS 9.2.1: Certificate validation
	// Note: ValidChain would be false if TLS handshake failed, so if we're here, it succeeded
	certInfo.ValidChain = true
	result.Standards.OWASPASVS9.Passed = append(result.Standards.OWASPASVS9.Passed, "9.2.1")

	// Check certificate expiry (PCI DSS 4.1)
	if certInfo.DaysUntilExpiry < 0 {
		issue := ComplianceIssue{
			Standard:    "PCI DSS 4.1 / OWASP ASVS 9.2.1",
			Requirement: "4.1-Certificate",
			Severity:    "critical",
			Description: "Certificate has expired",
			Remediation: "Renew the TLS certificate immediately.",
		}
		result.Issues = append(result.Issues, issue)
		result.Standards.PCIDSS41.Failed = append(result.Standards.PCIDSS41.Failed, "4.1-Certificate-Expiry")
		result.Standards.PCIDSS41.Compliant = false
	} else if certInfo.DaysUntilExpiry <= 30 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Certificate expires in %d days. Plan for renewal.", certInfo.DaysUntilExpiry))
	}

	// Check for self-signed certificates (warning, not failure for dev environments)
	if certInfo.SelfSigned {
		result.Recommendations = append(result.Recommendations,
			"Self-signed certificate detected. Use CA-signed certificates in production.")
	}

	// Check signature algorithm (PCI DSS 4.1 - minimum 112-bit strength)
	if strings.Contains(strings.ToLower(certInfo.SignatureAlg), "md5") ||
		strings.Contains(strings.ToLower(certInfo.SignatureAlg), "sha1") {
		issue := ComplianceIssue{
			Standard:    "PCI DSS 4.1",
			Requirement: "4.1-Certificate",
			Severity:    "high",
			Description: fmt.Sprintf("Weak signature algorithm: %s", certInfo.SignatureAlg),
			Remediation: "Use certificates with SHA-256 or stronger signature algorithms.",
		}
		result.Issues = append(result.Issues, issue)
		result.Standards.PCIDSS41.Failed = append(result.Standards.PCIDSS41.Failed, "4.1-Signature-Algorithm")
		result.Standards.PCIDSS41.Compliant = false
	} else {
		result.Standards.PCIDSS41.Passed = append(result.Standards.PCIDSS41.Passed, "4.1-Certificate-Valid")
	}

	// Check key size (PCI DSS 4.1 - minimum 2048-bit for RSA, 224-bit for ECC)
	if certInfo.KeySize > 0 {
		if strings.Contains(certInfo.PublicKeyAlg, "RSA") && certInfo.KeySize < 2048 {
			issue := ComplianceIssue{
				Standard:    "PCI DSS 4.1",
				Requirement: "4.1-Key-Strength",
				Severity:    "critical",
				Description: fmt.Sprintf("RSA key size too small: %d bits (minimum 2048 required)", certInfo.KeySize),
				Remediation: "Use RSA keys of at least 2048 bits (4096 bits recommended).",
			}
			result.Issues = append(result.Issues, issue)
			result.Standards.PCIDSS41.Failed = append(result.Standards.PCIDSS41.Failed, "4.1-Key-Size")
			result.Standards.PCIDSS41.Compliant = false
		} else if strings.Contains(certInfo.PublicKeyAlg, "ECDSA") && certInfo.KeySize < 224 {
			issue := ComplianceIssue{
				Standard:    "PCI DSS 4.1",
				Requirement: "4.1-Key-Strength",
				Severity:    "critical",
				Description: fmt.Sprintf("ECC key size too small: %d bits (minimum 224 required)", certInfo.KeySize),
				Remediation: "Use ECC keys of at least 224 bits (256 bits recommended).",
			}
			result.Issues = append(result.Issues, issue)
			result.Standards.PCIDSS41.Failed = append(result.Standards.PCIDSS41.Failed, "4.1-Key-Size")
			result.Standards.PCIDSS41.Compliant = false
		}
	}
}

// tlsVersionString converts TLS version constant to string
func tlsVersionString(version uint16) string {
	switch version {
	case versionSSL30:
		return "SSL 3.0"
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// cipherSuiteString converts cipher suite constant to string
func cipherSuiteString(suite uint16) string {
	// Check strong suites first
	if name, ok := strongCipherSuites[suite]; ok {
		return name
	}

	// Check weak suites
	if name, ok := weakCipherSuites[suite]; ok {
		return name
	}

	// Use tls.CipherSuiteName for Go 1.14+
	name := tls.CipherSuiteName(suite)
	if name != "" {
		return name
	}

	return fmt.Sprintf("Unknown (0x%04x)", suite)
}
