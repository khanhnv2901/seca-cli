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
package checker

