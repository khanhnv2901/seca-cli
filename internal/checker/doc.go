// Package checker defines the core SECA-CLI checking framework.
//
// Architecture overview:
//
//   - Checkers implement the Checker interface (Check + Name) for specific
//     protocols such as HTTP, DNS, or Network. ExternalChecker adapts community
//     plugins that emit JSON CheckResult payloads.
//   - Runner coordinates concurrent execution with rate limiting, invoking
//     a shared AuditFunc per target so every run produces consistent evidence.
//   - Shared result structs (CheckResult, SecurityHeadersResult, TLSComplianceResult,
//     NetworkSecurityResult, etc.) model the telemetry stored in results.json and
//     consumed by reports.
//   - Helper utilities (ParseTarget, AnalyzeSecurityHeaders, AnalyzeTLSCompliance,
//     and so on) are factored here so CLI commands in cmd/ simply instantiate
//     a checker and feed it into the runner.
//
// This layout keeps protocol logic internal while allowing cmd/ to treat every
// checker (built-in or plugin) uniformly.
//
// Network Security Checks:
//
//   NetworkChecker provides comprehensive network security assessments:
//   - Open port scanning with configurable port lists and risk classification
//   - Subdomain takeover vulnerability detection with 20+ provider fingerprints
//   - Concurrent port scanning with worker pools for efficient checking
//   - Banner grabbing for service identification
//   - Risk-based recommendations (critical/high/medium/low/info)
//
// Advanced TLS Checks:
//
//   TLS compliance analysis has been enhanced with:
//   - Mixed Content Detection: Identifies HTTP resources on HTTPS pages
//     (scripts, stylesheets, images, media, iframes) with severity classification
//   - OCSP Stapling: Verifies if server supports OCSP stapling for improved
//     certificate revocation checking performance and privacy (OWASP ASVS 9.2.4)
//
// Client-Side Security Analysis:
//
//   Comprehensive client-side security assessments include:
//   - Vulnerable JavaScript Libraries: Detects outdated libraries with known CVEs
//     (jQuery, AngularJS, Lodash, Moment.js, Bootstrap, React, Vue)
//     Tracks specific vulnerabilities: CVE-2020-11022, CVE-2020-11023, CVE-2019-10768,
//     CVE-2019-10744, CVE-2022-24785, CVE-2019-8331
//   - Anti-CSRF Protection: Analyzes CSRF tokens in meta tags, form inputs, headers,
//     and cookies. Evaluates protection strength (none/weak/moderate/strong) based on
//     synchronizer token pattern, double-submit cookies, and SameSite cookie attributes
//   - Trusted Types: Checks for DOM-based XSS prevention via CSP require-trusted-types-for
//     directive (modern browser protection against injection attacks)
//   - Severity-based Recommendations: Provides actionable remediation guidance with
//     CVSS scores and upgrade paths for vulnerable dependencies
//
package checker

