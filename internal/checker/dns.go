package checker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSChecker performs DNS resolution checks
type DNSChecker struct {
	Timeout    time.Duration
	NameServer []string // Optional custom nameservers
}

// Check performs DNS resolution checks on the target
func (d *DNSChecker) Check(ctx context.Context, target string) CheckResult {
	result := CheckResult{
		Target:     target,
		CheckedAt:  time.Now().UTC(),
		DNSRecords: make(map[string]interface{}),
	}

	// Remove protocol prefix if present
	host := strings.TrimPrefix(target, "http://")
	host = strings.TrimPrefix(host, "https://")
	host = strings.Split(host, "/")[0] // Remove path
	host = strings.Split(host, ":")[0] // Remove port

	// Create resolver
	resolver := &net.Resolver{
		PreferGo: true,
	}

	// If custom nameservers provided, use them
	if len(d.NameServer) > 0 {
		dialer := &net.Dialer{
			Timeout: d.Timeout,
		}
		resolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			// Use first nameserver for now
			return dialer.DialContext(ctx, network, d.NameServer[0])
		}
	}

	// Create context with timeout
	lookupCtx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	// Perform A record lookup
	aRecords, err := resolver.LookupHost(lookupCtx, host)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("DNS lookup failed: %v", err)
		return result
	}

	if len(aRecords) == 0 {
		result.Status = "error"
		result.Error = "no A records found"
		return result
	}

	result.DNSRecords["a_records"] = aRecords
	result.Status = "ok"
	result.Notes = fmt.Sprintf("%d A record(s) found", len(aRecords))

	// Lookup AAAA records (ipv6)
	lookupCtx2, cancel2 := context.WithTimeout(ctx, d.Timeout)
	defer cancel2()

	aaaaRecords, err := resolver.LookupIP(lookupCtx2, "ipv6", host)
	if err == nil && len(aaaaRecords) > 0 {
		ipv6Addrs := make([]string, 0, len(aaaaRecords))
		for _, ip := range aaaaRecords {
			ipv6Addrs = append(ipv6Addrs, ip.String())
		}
		result.DNSRecords["aaaa_records"] = ipv6Addrs
		result.Notes += fmt.Sprintf(", %d AAAA record(s) found", len(aaaaRecords))
	}

	// Lookup CNAME records
	lookupCtx3, cancel3 := context.WithTimeout(ctx, d.Timeout)
	defer cancel3()

	cname, err := resolver.LookupCNAME(lookupCtx3, host)
	if err == nil && cname != host && cname != host+"." {
		result.DNSRecords["cname"] = cname
		result.Notes += ", CNAME found"
	}

	// Lookup MX records
	lookupCtx4, cancel4 := context.WithTimeout(ctx, d.Timeout)
	defer cancel4()

	mxRecords, err := resolver.LookupMX(lookupCtx4, host)
	if err == nil && len(mxRecords) > 0 {
		mxHosts := make([]map[string]interface{}, 0, len(mxRecords))
		for _, mx := range mxRecords {
			mxHosts = append(mxHosts, map[string]interface{}{
				"host":     mx.Host,
				"priority": mx.Pref,
			})
		}
		result.DNSRecords["mx_records"] = mxHosts
		result.Notes += fmt.Sprintf(", %d MX recrod(s) found", len(mxRecords))
	}

	// Look up NS records
	lookupCtx5, cancel5 := context.WithTimeout(ctx, d.Timeout)
	defer cancel5()

	nsRecords, err := resolver.LookupNS(lookupCtx5, host)
	if err == nil && len(nsRecords) > 0 {
		nsHosts := make([]string, 0, len(nsRecords))
		for _, ns := range nsRecords {
			nsHosts = append(nsHosts, ns.Host)
		}
		result.DNSRecords["ns_records"] = nsHosts
		result.Notes += fmt.Sprintf(", %d NS record(s) found", len(nsRecords))
	}

	// Lookup TXT records
	lookupCtx6, cancel6 := context.WithTimeout(ctx, d.Timeout)
	defer cancel6()

	txtRecords, err := resolver.LookupTXT(lookupCtx6, host)
	if err == nil && len(txtRecords) > 0 {
		result.DNSRecords["txt_records"] = txtRecords
		result.Notes += fmt.Sprintf(", %d TXT record(s) found", len(txtRecords))
	}

	// Reverse DNS lookup (PTR records) for first A record
	if len(aRecords) > 0 {
		lookupCtx7, cancel7 := context.WithTimeout(ctx, d.Timeout)
		defer cancel7()

		ptrRecords, err := resolver.LookupAddr(lookupCtx7, aRecords[0])
		if err == nil && len(ptrRecords) > 0 {
			result.DNSRecords["ptr_records"] = ptrRecords
			result.Notes += ", PTR record(s) found"
		}
	}

	return result
}

func (d *DNSChecker) Name() string {
	return "check dns"
}
