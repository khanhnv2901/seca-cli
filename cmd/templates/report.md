# Engagement Report: {{.Metadata.EngagementName}}

**Generated:** {{.GeneratedAt}}

## Metadata

- **Engagement ID:** {{.Metadata.EngagementID}}
- **Engagement Name:** {{.Metadata.EngagementName}}
- **Owner:** {{.Metadata.Owner}}
- **Operator:** {{.Metadata.Operator}}
- **Started At:** {{.StartedAt}}
- **Completed At:** {{.CompletedAt}}
- **Duration:** {{.Duration}}
- **Total Targets:** {{.Metadata.TotalTargets}}
{{if .Metadata.AuditHash}}- **Audit Hash (SHA256):** `{{.Metadata.AuditHash}}`
{{end}}

## Summary

- **Successful:** {{.SuccessCount}}
- **Failed:** {{.ErrorCount}}
- **Success Rate:** {{.SuccessRate}}%

## Results Overview

| Target | Status | HTTP Status | Server | TLS Expiry | Notes |
|--------|--------|-------------|--------|------------|-------|
{{range .Results}}| {{.Target}} | {{.Status}} | {{if .HTTPStatus}}{{.HTTPStatus}}{{end}} | {{.ServerHeader}} | {{.TLSExpiry}} | {{if .Notes}}{{.Notes}}{{else}}-{{end}} |
{{end}}

## Detailed Security Analysis
{{range $index, $result := .Results}}
### {{add $index 1}}. {{$result.Target}}

#### Basic Information

- **Status:** {{$result.Status}}
{{if $result.HTTPStatus}}- **HTTP Status:** {{$result.HTTPStatus}}
{{end}}{{if $result.ServerHeader}}- **Server:** {{$result.ServerHeader}}
{{end}}{{if gt $result.ResponseTime 0.0}}- **Response Time:** {{printf "%.2f" $result.ResponseTime}} ms
{{end}}{{if $result.Notes}}- **Notes:** {{$result.Notes}}
{{end}}{{if $result.Error}}- **Error:** {{$result.Error}}
{{end}}
{{if $result.SecurityHeaders}}#### Security Headers Analysis

**Overall Score:** {{$result.SecurityHeaders.Score}}/{{$result.SecurityHeaders.MaxScore}} (**Grade: {{$result.SecurityHeaders.Grade}}**)

- **Headers Present:** {{headersPresentCount $result.SecurityHeaders}}/8
- **Headers Missing:** {{len $result.SecurityHeaders.Missing}}

**Header Details:**
{{range $name, $header := $result.SecurityHeaders.Headers}}
{{if $header.Present}}- ✅ **{{$name}}** (Score: {{$header.Score}}/{{$header.MaxScore}})
{{if $header.Value}}  - Value: `{{$header.Value}}`
{{end}}{{if $header.Issues}}  - Issues:
{{range $header.Issues}}    - {{.}}
{{end}}{{end}}{{if $header.Recommendation}}  - Recommendation: {{$header.Recommendation}}
{{end}}{{else}}- ❌ **{{$name}}** (Score: {{$header.Score}}/{{$header.MaxScore}}, Severity: {{$header.Severity}})
{{if $header.Recommendation}}  - Recommendation: {{$header.Recommendation}}
{{end}}{{end}}{{end}}
{{if $result.SecurityHeaders.Warnings}}**Warnings:**
{{range $result.SecurityHeaders.Warnings}}
- ⚠️ {{.}}
{{end}}
{{end}}{{if hasHighSeverityMissing $result.SecurityHeaders}}**Priority Recommendations:**

{{if highSeverityMissing $result.SecurityHeaders}}🔴 **High Priority:**
{{range highSeverityMissing $result.SecurityHeaders}}- Implement {{.}}
{{end}}
{{end}}{{if mediumSeverityMissing $result.SecurityHeaders}}🟡 **Medium Priority:**
{{range mediumSeverityMissing $result.SecurityHeaders}}- Implement {{.}}
{{end}}
{{end}}{{end}}{{end}}
{{if $result.TLSCompliance}}#### TLS Compliance Analysis

**Overall Status:** {{if $result.TLSCompliance.Compliant}}✅ **COMPLIANT**{{else}}❌ **NON-COMPLIANT**{{end}}

**Configuration:**

- **TLS Version:** {{$result.TLSCompliance.TLSVersion}}
- **Cipher Suite:** {{$result.TLSCompliance.CipherSuite}}

**Standards Compliance:**

- {{if $result.TLSCompliance.Standards.OWASPASVS9.Compliant}}✅{{else}}❌{{end}} **OWASP ASVS v9** (Level: {{$result.TLSCompliance.Standards.OWASPASVS9.Level}})
{{if $result.TLSCompliance.Standards.OWASPASVS9.Passed}}  - Passed Controls: {{join $result.TLSCompliance.Standards.OWASPASVS9.Passed ", "}}
{{end}}{{if $result.TLSCompliance.Standards.OWASPASVS9.Failed}}  - Failed Controls: {{join $result.TLSCompliance.Standards.OWASPASVS9.Failed ", "}}
{{end}}- {{if $result.TLSCompliance.Standards.PCIDSS41.Compliant}}✅{{else}}❌{{end}} **PCI DSS 4.1**
{{if $result.TLSCompliance.Standards.PCIDSS41.Passed}}  - Passed Controls: {{join $result.TLSCompliance.Standards.PCIDSS41.Passed ", "}}
{{end}}{{if $result.TLSCompliance.Standards.PCIDSS41.Failed}}  - Failed Controls: {{join $result.TLSCompliance.Standards.PCIDSS41.Failed ", "}}
{{end}}- {{if $result.TLSCompliance.Standards.NIST80052r2.Compliant}}✅{{else}}❌{{end}} **NIST SP 800-52r2** (Level: {{$result.TLSCompliance.Standards.NIST80052r2.Level}})
{{if $result.TLSCompliance.Standards.NIST80052r2.Passed}}  - Passed Controls: {{join $result.TLSCompliance.Standards.NIST80052r2.Passed ", "}}
{{end}}{{if $result.TLSCompliance.Standards.NIST80052r2.Failed}}  - Failed Controls: {{join $result.TLSCompliance.Standards.NIST80052r2.Failed ", "}}
{{end}}
{{if $result.TLSCompliance.CertificateInfo}}**Certificate Information:**

- **Subject:** {{$result.TLSCompliance.CertificateInfo.Subject}}
- **Issuer:** {{$result.TLSCompliance.CertificateInfo.Issuer}}
- **Valid From:** {{$result.TLSCompliance.CertificateInfo.NotBefore}}
- **Valid Until:** {{$result.TLSCompliance.CertificateInfo.NotAfter}}
- **Days Until Expiry:** {{$result.TLSCompliance.CertificateInfo.DaysUntilExpiry}}
{{if $result.TLSCompliance.CertificateInfo.DNSNames}}- **DNS Names:** {{join $result.TLSCompliance.CertificateInfo.DNSNames ", "}}
{{end}}- **Self-Signed:** {{$result.TLSCompliance.CertificateInfo.SelfSigned}}
- **Valid Chain:** {{$result.TLSCompliance.CertificateInfo.ValidChain}}
- **Signature Algorithm:** {{$result.TLSCompliance.CertificateInfo.SignatureAlg}}
- **Public Key Algorithm:** {{$result.TLSCompliance.CertificateInfo.PublicKeyAlg}}
- **Key Size:** {{$result.TLSCompliance.CertificateInfo.KeySize}} bits
- **Chain Depth:** {{$result.TLSCompliance.CertificateInfo.ChainDepth}}
{{if $result.TLSCompliance.CertificateInfo.ChainSubjects}}- **Certificate Chain:**
{{range $i, $subject := $result.TLSCompliance.CertificateInfo.ChainSubjects}}  {{add $i 1}}. {{$subject}}
{{end}}{{end}}
{{end}}{{if $result.TLSCompliance.Recommendations}}**Recommendations:**
{{range $result.TLSCompliance.Recommendations}}
- {{.}}
{{end}}
{{end}}{{end}}
{{if $result.DNSRecords}}#### DNS Records
{{if index $result.DNSRecords "a_records"}}
**A Records (IPv4):**
{{range index $result.DNSRecords "a_records"}}
- {{.}}
{{end}}
{{end}}{{if index $result.DNSRecords "aaaa_records"}}**AAAA Records (IPv6):**
{{range index $result.DNSRecords "aaaa_records"}}
- {{.}}
{{end}}
{{end}}{{if index $result.DNSRecords "cname_records"}}**CNAME Records:**
{{range index $result.DNSRecords "cname_records"}}
- {{.}}
{{end}}
{{end}}{{if index $result.DNSRecords "mx_records"}}**MX Records (Mail Servers):**
{{range index $result.DNSRecords "mx_records"}}
- {{.}}
{{end}}
{{end}}{{if index $result.DNSRecords "ns_records"}}**NS Records (Name Servers):**
{{range index $result.DNSRecords "ns_records"}}
- {{.}}
{{end}}
{{end}}{{if index $result.DNSRecords "txt_records"}}**TXT Records:**
{{range index $result.DNSRecords "txt_records"}}
- {{.}}
{{end}}
{{end}}{{end}}
---
{{end}}

*Report generated by seca-cli on {{.FooterDate}}*
