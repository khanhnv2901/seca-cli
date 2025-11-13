package checker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// NetworkSecurityResult contains network security analysis results
type NetworkSecurityResult struct {
	OpenPorts         []PortInfo       `json:"open_ports,omitempty"`
	SubdomainTakeover *SubdomainCheck  `json:"subdomain_takeover,omitempty"`
	PortScanDuration  float64          `json:"port_scan_duration_ms,omitempty"`
	Issues            []string         `json:"issues,omitempty"`
	Recommendations   []string         `json:"recommendations,omitempty"`
}

// PortInfo contains information about an open port
type PortInfo struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`     // "tcp" or "udp"
	State       string `json:"state"`        // "open", "closed", "filtered"
	Service     string `json:"service"`      // Common service name (e.g., "http", "https", "ssh")
	Banner      string `json:"banner,omitempty"`
	Risk        string `json:"risk"`         // "critical", "high", "medium", "low", "info"
	Description string `json:"description,omitempty"`
}

// SubdomainCheck contains subdomain takeover vulnerability analysis
type SubdomainCheck struct {
	Vulnerable      bool     `json:"vulnerable"`
	CNAME           string   `json:"cname,omitempty"`
	Provider        string   `json:"provider,omitempty"`        // e.g., "AWS S3", "GitHub Pages", "Heroku"
	Fingerprint     string   `json:"fingerprint,omitempty"`     // Detection fingerprint
	Confidence      string   `json:"confidence"`                // "high", "medium", "low"
	ResolvedIPs     []string `json:"resolved_ips,omitempty"`
	HTTPStatusCode  int      `json:"http_status_code,omitempty"`
	ErrorMessage    string   `json:"error_message,omitempty"`
	Recommendation  string   `json:"recommendation,omitempty"`
}

// NetworkChecker performs network security checks
type NetworkChecker struct {
	Timeout         time.Duration
	PortScanTimeout time.Duration
	EnablePortScan  bool
	CommonPorts     []int  // Ports to scan (e.g., [80, 443, 22, 21, 25, 3306, 5432])
	MaxPortWorkers  int    // Concurrent port scans
}

// Check performs network security checks on the target
func (n *NetworkChecker) Check(ctx context.Context, target string) CheckResult {
	result := CheckResult{
		Target:    target,
		CheckedAt: time.Now().UTC(),
		Status:    "ok",
	}

	// Initialize network security result
	netSec := &NetworkSecurityResult{
		OpenPorts: []PortInfo{},
		Issues:    []string{},
		Recommendations: []string{},
	}

	// Extract hostname
	host := ExtractHost(target)

	// 1. Check for subdomain takeover vulnerability
	subdomainCheck := n.checkSubdomainTakeover(ctx, host)
	netSec.SubdomainTakeover = subdomainCheck

	if subdomainCheck.Vulnerable {
		netSec.Issues = append(netSec.Issues,
			fmt.Sprintf("Subdomain takeover vulnerability detected (Provider: %s, Confidence: %s)",
				subdomainCheck.Provider, subdomainCheck.Confidence))
		netSec.Recommendations = append(netSec.Recommendations, subdomainCheck.Recommendation)
		result.Notes = "CRITICAL: Subdomain takeover vulnerability detected"
	}

	// 2. Perform port scan if enabled
	if n.EnablePortScan {
		startTime := time.Now()
		openPorts := n.scanPorts(ctx, host)
		netSec.PortScanDuration = time.Since(startTime).Seconds() * 1000
		netSec.OpenPorts = openPorts

		// Analyze port risks
		n.analyzePortRisks(netSec)

		if len(openPorts) > 0 {
			if result.Notes != "" {
				result.Notes += "; "
			}
			result.Notes += fmt.Sprintf("%d open port(s) found", len(openPorts))
		}
	}

	result.NetworkSecurity = netSec
	return result
}

// checkSubdomainTakeover detects potential subdomain takeover vulnerabilities
func (n *NetworkChecker) checkSubdomainTakeover(ctx context.Context, host string) *SubdomainCheck {
	check := &SubdomainCheck{
		Vulnerable: false,
		Confidence: "low",
	}

	// Create resolver
	resolver := &net.Resolver{
		PreferGo: true,
	}

	// Create context with timeout
	lookupCtx, cancel := context.WithTimeout(ctx, n.Timeout)
	defer cancel()

	// 1. Check CNAME record
	cname, err := resolver.LookupCNAME(lookupCtx, host)
	if err != nil {
		check.ErrorMessage = fmt.Sprintf("CNAME lookup failed: %v", err)
		return check
	}

	// Normalize CNAME (remove trailing dot)
	cname = strings.TrimSuffix(cname, ".")
	check.CNAME = cname

	// If no CNAME or CNAME points to itself, no takeover risk
	if cname == "" || cname == host {
		return check
	}

	// 2. Try to resolve the CNAME target
	ipLookupCtx, ipCancel := context.WithTimeout(ctx, n.Timeout)
	defer ipCancel()

	ips, err := resolver.LookupHost(ipLookupCtx, cname)
	if err != nil {
		// CNAME exists but doesn't resolve - potential takeover
		check.Vulnerable = true
		check.Confidence = "medium"
		check.ErrorMessage = fmt.Sprintf("CNAME target does not resolve: %v", err)

		// Detect provider from CNAME pattern
		check.Provider = detectProvider(cname)
		check.Fingerprint = "CNAME exists but target does not resolve"
		check.Recommendation = fmt.Sprintf(
			"The subdomain has a CNAME pointing to %s which does not resolve. "+
			"This may indicate a subdomain takeover vulnerability. "+
			"Verify that the %s resource exists and is properly configured.",
			cname, check.Provider)

		// Increase confidence if we detect a known vulnerable provider
		if check.Provider != "Unknown" {
			check.Confidence = "high"
		}

		return check
	}

	check.ResolvedIPs = ips

	// 3. Check HTTP response for takeover fingerprints
	httpCheck := n.checkHTTPFingerprints(ctx, host, cname, check.Provider)
	if httpCheck.Vulnerable {
		check.Vulnerable = true
		check.Confidence = httpCheck.Confidence
		check.HTTPStatusCode = httpCheck.HTTPStatusCode
		check.Fingerprint = httpCheck.Fingerprint
		check.Provider = httpCheck.Provider
		check.Recommendation = httpCheck.Recommendation
	}

	return check
}

// checkHTTPFingerprints checks HTTP responses for subdomain takeover fingerprints
func (n *NetworkChecker) checkHTTPFingerprints(ctx context.Context, host, cname, detectedProvider string) *SubdomainCheck {
	check := &SubdomainCheck{
		Vulnerable: false,
		Confidence: "low",
		Provider:   detectedProvider,
	}

	// Try HTTPS first, then HTTP
	schemes := []string{"https", "http"}

	for _, scheme := range schemes {
		url := fmt.Sprintf("%s://%s", scheme, host)

		client := &http.Client{
			Timeout: n.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		check.HTTPStatusCode = resp.StatusCode

		// Read response body for fingerprint matching
		body := make([]byte, 8192) // Read first 8KB
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])

		// Check for known takeover fingerprints
		fingerprints := getTakeoverFingerprints()

		for provider, patterns := range fingerprints {
			for _, pattern := range patterns {
				if strings.Contains(bodyStr, pattern) || strings.Contains(resp.Header.Get("Server"), pattern) {
					check.Vulnerable = true
					check.Confidence = "high"
					check.Provider = provider
					check.Fingerprint = pattern
					check.Recommendation = fmt.Sprintf(
						"The subdomain shows signs of being claimable on %s. "+
						"Detected fingerprint: '%s'. "+
						"Verify ownership of the %s resource or remove the DNS record.",
						provider, pattern, provider)
					return check
				}
			}
		}

		break // If we got a response, don't try other schemes
	}

	return check
}

// getTakeoverFingerprints returns known subdomain takeover fingerprints
func getTakeoverFingerprints() map[string][]string {
	return map[string][]string{
		"GitHub Pages": {
			"There isn't a GitHub Pages site here",
			"For root URLs (like http://example.com/) you must provide an index.html file",
		},
		"AWS S3": {
			"NoSuchBucket",
			"The specified bucket does not exist",
		},
		"Heroku": {
			"No such app",
			"herokucdn.com/error-pages/no-such-app.html",
		},
		"Azure": {
			"404 Web Site not found",
			"Error 404 - Web app not found",
		},
		"Shopify": {
			"Sorry, this shop is currently unavailable",
			"Only one step left!",
		},
		"Tumblr": {
			"Whatever you were looking for doesn't currently exist at this address",
			"There's nothing here",
		},
		"WordPress.com": {
			"Do you want to register",
			"doesn't exist",
		},
		"Ghost": {
			"The thing you were looking for is no longer here",
		},
		"Bitbucket": {
			"Repository not found",
		},
		"Fastly": {
			"Fastly error: unknown domain",
		},
		"Pantheon": {
			"404 error unknown site!",
		},
		"Zendesk": {
			"Help Center Closed",
		},
		"UserVoice": {
			"This UserVoice subdomain is currently available",
		},
		"Surge.sh": {
			"project not found",
		},
		"Intercom": {
			"This page is reserved for artistic dogs",
			"Uh oh. That page doesn't exist",
		},
		"Webflow": {
			"The page you are looking for doesn't exist or has been moved",
		},
		"Cargo Collective": {
			"If you're moving your domain away from Cargo",
		},
		"StatusPage": {
			"You are being",
			"redirected",
		},
		"Readme.io": {
			"Project doesnt exist... yet!",
		},
	}
}

// detectProvider detects the service provider from CNAME pattern
func detectProvider(cname string) string {
	patterns := map[string][]string{
		"GitHub Pages":       {"github.io", "githubusercontent.com"},
		"AWS S3":             {".s3.amazonaws.com", ".s3-website"},
		"AWS CloudFront":     {"cloudfront.net"},
		"Heroku":             {"herokuapp.com", "herokussl.com"},
		"Azure":              {"azurewebsites.net", "cloudapp.azure.com", "azure.com"},
		"Shopify":            {"myshopify.com"},
		"Tumblr":             {"tumblr.com"},
		"WordPress.com":      {"wordpress.com"},
		"Ghost":              {"ghost.io"},
		"Bitbucket":          {"bitbucket.io"},
		"Fastly":             {"fastly.net"},
		"Pantheon":           {"pantheonsite.io"},
		"Zendesk":            {"zendesk.com"},
		"UserVoice":          {"uservoice.com"},
		"Surge.sh":           {"surge.sh"},
		"Intercom":           {"intercom.io", "intercomcdn.com"},
		"Webflow":            {"webflow.io"},
		"Cargo Collective":   {"cargocollective.com"},
		"StatusPage":         {"statuspage.io"},
		"Readme.io":          {"readme.io"},
		"Netlify":            {"netlify.app", "netlify.com"},
		"Vercel":             {"vercel.app", "vercel.com"},
		"AWS Elastic Beanstalk": {"elasticbeanstalk.com"},
		"DigitalOcean Spaces": {"digitaloceanspaces.com"},
	}

	cnameLower := strings.ToLower(cname)

	for provider, patterns := range patterns {
		for _, pattern := range patterns {
			if strings.Contains(cnameLower, pattern) {
				return provider
			}
		}
	}

	return "Unknown"
}

// scanPorts performs a port scan on common ports
func (n *NetworkChecker) scanPorts(ctx context.Context, host string) []PortInfo {
	// Use default common ports if not specified
	ports := n.CommonPorts
	if len(ports) == 0 {
		ports = []int{
			21,   // FTP
			22,   // SSH
			23,   // Telnet
			25,   // SMTP
			53,   // DNS
			80,   // HTTP
			110,  // POP3
			143,  // IMAP
			443,  // HTTPS
			445,  // SMB
			3306, // MySQL
			3389, // RDP
			5432, // PostgreSQL
			5900, // VNC
			6379, // Redis
			8080, // HTTP Alt
			8443, // HTTPS Alt
			27017, // MongoDB
		}
	}

	maxWorkers := n.MaxPortWorkers
	if maxWorkers == 0 {
		maxWorkers = 10 // Default concurrency
	}

	// Create worker pool
	portChan := make(chan int, len(ports))
	resultChan := make(chan *PortInfo, len(ports))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range portChan {
				if portInfo := n.checkPort(ctx, host, port); portInfo != nil {
					resultChan <- portInfo
				}
			}
		}()
	}

	// Send ports to workers
	go func() {
		for _, port := range ports {
			select {
			case portChan <- port:
			case <-ctx.Done():
				break
			}
		}
		close(portChan)
	}()

	// Wait for workers and close result channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	openPorts := []PortInfo{}
	for portInfo := range resultChan {
		openPorts = append(openPorts, *portInfo)
	}

	return openPorts
}

// checkPort checks if a specific port is open
func (n *NetworkChecker) checkPort(ctx context.Context, host string, port int) *PortInfo {
	timeout := n.PortScanTimeout
	if timeout == 0 {
		timeout = 2 * time.Second // Default port scan timeout
	}

	address := fmt.Sprintf("%s:%d", host, port)

	// Use context with timeout
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		// Port is closed or filtered
		return nil
	}
	defer conn.Close()

	// Port is open
	portInfo := &PortInfo{
		Port:     port,
		Protocol: "tcp",
		State:    "open",
		Service:  getServiceName(port),
		Risk:     getPortRisk(port),
	}

	// Try to grab banner (with short timeout)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	banner := make([]byte, 512)
	bytesRead, readErr := conn.Read(banner)
	if readErr == nil && bytesRead > 0 {
		portInfo.Banner = strings.TrimSpace(string(banner[:bytesRead]))
	}

	return portInfo
}

// getServiceName returns common service name for a port
func getServiceName(port int) string {
	services := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		443:   "https",
		445:   "smb",
		3306:  "mysql",
		3389:  "rdp",
		5432:  "postgresql",
		5900:  "vnc",
		6379:  "redis",
		8080:  "http-alt",
		8443:  "https-alt",
		27017: "mongodb",
	}

	if service, ok := services[port]; ok {
		return service
	}
	return "unknown"
}

// getPortRisk assigns risk level to open ports
func getPortRisk(port int) string {
	criticalPorts := []int{23, 3389, 5900} // Telnet, RDP, VNC
	highPorts := []int{21, 22, 445, 3306, 5432, 6379, 27017} // FTP, SSH, SMB, Databases
	mediumPorts := []int{25, 110, 143, 8080, 8443} // Mail, HTTP alts

	for _, p := range criticalPorts {
		if port == p {
			return "critical"
		}
	}

	for _, p := range highPorts {
		if port == p {
			return "high"
		}
	}

	for _, p := range mediumPorts {
		if port == p {
			return "medium"
		}
	}

	// Standard web ports
	if port == 80 || port == 443 {
		return "low"
	}

	return "info"
}

// analyzePortRisks adds security recommendations based on open ports
func (n *NetworkChecker) analyzePortRisks(netSec *NetworkSecurityResult) {
	criticalCount := 0
	highCount := 0

	for i := range netSec.OpenPorts {
		port := &netSec.OpenPorts[i]
		switch port.Risk {
		case "critical":
			criticalCount++
			port.Description = fmt.Sprintf("CRITICAL: Port %d (%s) should not be exposed to the internet",
				port.Port, port.Service)
		case "high":
			highCount++
			port.Description = fmt.Sprintf("HIGH RISK: Port %d (%s) exposed - ensure proper authentication and encryption",
				port.Port, port.Service)
		case "medium":
			port.Description = fmt.Sprintf("MEDIUM RISK: Port %d (%s) exposed - review security configuration",
				port.Port, port.Service)
		case "low":
			port.Description = fmt.Sprintf("LOW RISK: Port %d (%s) is a standard web port",
				port.Port, port.Service)
		case "info":
			port.Description = fmt.Sprintf("INFO: Port %d (%s) is open",
				port.Port, port.Service)
		}
	}

	if criticalCount > 0 {
		netSec.Issues = append(netSec.Issues,
			fmt.Sprintf("%d critical port(s) exposed (Telnet/RDP/VNC)", criticalCount))
		netSec.Recommendations = append(netSec.Recommendations,
			"Close or firewall critical ports. Use VPN for remote access instead of direct exposure.")
	}

	if highCount > 0 {
		netSec.Issues = append(netSec.Issues,
			fmt.Sprintf("%d high-risk port(s) exposed (SSH/Database/SMB)", highCount))
		netSec.Recommendations = append(netSec.Recommendations,
			"Restrict database and administrative ports to trusted IPs only. Use strong authentication.")
	}
}

// Name returns the checker name
func (n *NetworkChecker) Name() string {
	return "check network"
}
