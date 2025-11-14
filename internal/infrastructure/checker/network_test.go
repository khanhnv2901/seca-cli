package checker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNetworkChecker_Name(t *testing.T) {
	checker := &NetworkChecker{}
	if got := checker.Name(); got != "check network" {
		t.Errorf("NetworkChecker.Name() = %v, want %v", got, "check network")
	}
}

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		port    int
		want    string
	}{
		{80, "http"},
		{443, "https"},
		{22, "ssh"},
		{3306, "mysql"},
		{5432, "postgresql"},
		{6379, "redis"},
		{27017, "mongodb"},
		{9999, "unknown"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("port_%d", tt.port), func(t *testing.T) {
			if got := getServiceName(tt.port); got != tt.want {
				t.Errorf("getServiceName(%d) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}

func TestGetPortRisk(t *testing.T) {
	tests := []struct {
		port int
		want string
	}{
		{23, "critical"},     // Telnet
		{3389, "critical"},   // RDP
		{5900, "critical"},   // VNC
		{22, "high"},         // SSH
		{3306, "high"},       // MySQL
		{5432, "high"},       // PostgreSQL
		{8080, "medium"},     // HTTP alt
		{80, "low"},          // HTTP
		{443, "low"},         // HTTPS
		{12345, "info"},      // Unknown
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("port_%d", tt.port), func(t *testing.T) {
			if got := getPortRisk(tt.port); got != tt.want {
				t.Errorf("getPortRisk(%d) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		cname string
		want  string
	}{
		{"myapp.github.io", "GitHub Pages"},
		{"example.s3.amazonaws.com", "AWS S3"},
		{"myapp.herokuapp.com", "Heroku"},
		{"mysite.azurewebsites.net", "Azure"},
		{"shop.myshopify.com", "Shopify"},
		{"blog.tumblr.com", "Tumblr"},
		{"site.wordpress.com", "WordPress.com"},
		{"app.ghost.io", "Ghost"},
		{"code.bitbucket.io", "Bitbucket"},
		{"cdn.fastly.net", "Fastly"},
		{"site.pantheonsite.io", "Pantheon"},
		{"support.zendesk.com", "Zendesk"},
		{"feedback.uservoice.com", "UserVoice"},
		{"mysite.surge.sh", "Surge.sh"},
		{"chat.intercom.io", "Intercom"},
		{"portfolio.webflow.io", "Webflow"},
		{"art.cargocollective.com", "Cargo Collective"},
		{"status.statuspage.io", "StatusPage"},
		{"docs.readme.io", "Readme.io"},
		{"app.netlify.app", "Netlify"},
		{"myapp.vercel.app", "Vercel"},
		{"env.elasticbeanstalk.com", "AWS Elastic Beanstalk"},
		{"files.digitaloceanspaces.com", "DigitalOcean Spaces"},
		{"unknown-provider.example.com", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.cname, func(t *testing.T) {
			if got := detectProvider(tt.cname); got != tt.want {
				t.Errorf("detectProvider(%q) = %v, want %v", tt.cname, got, tt.want)
			}
		})
	}
}

func TestGetTakeoverFingerprints(t *testing.T) {
	fingerprints := getTakeoverFingerprints()

	// Verify expected providers are present
	expectedProviders := []string{
		"GitHub Pages",
		"AWS S3",
		"Heroku",
		"Azure",
		"Shopify",
		"Tumblr",
		"WordPress.com",
		"Ghost",
		"Bitbucket",
		"Fastly",
		"Pantheon",
		"Zendesk",
		"UserVoice",
		"Surge.sh",
		"Intercom",
		"Webflow",
		"Cargo Collective",
		"StatusPage",
		"Readme.io",
	}

	for _, provider := range expectedProviders {
		if _, ok := fingerprints[provider]; !ok {
			t.Errorf("Expected provider %q not found in fingerprints", provider)
		}
	}

	// Verify each provider has at least one fingerprint
	for provider, patterns := range fingerprints {
		if len(patterns) == 0 {
			t.Errorf("Provider %q has no fingerprint patterns", provider)
		}
	}
}

func TestCheckHTTPFingerprints_GitHubPages(t *testing.T) {
	// Create test server that returns GitHub Pages error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("There isn't a GitHub Pages site here."))
	}))
	defer server.Close()

	checker := &NetworkChecker{
		Timeout: 5 * time.Second,
	}

	// Extract host from test server URL
	host := server.URL[7:] // Remove "http://"

	result := checker.checkHTTPFingerprints(context.Background(), host, "example.github.io", "GitHub Pages")

	if !result.Vulnerable {
		t.Error("Expected vulnerable=true for GitHub Pages fingerprint")
	}

	if result.Provider != "GitHub Pages" {
		t.Errorf("Expected provider=GitHub Pages, got %q", result.Provider)
	}

	if result.Confidence != "high" {
		t.Errorf("Expected confidence=high, got %q", result.Confidence)
	}
}

func TestCheckHTTPFingerprints_AWSS3(t *testing.T) {
	// Create test server that returns AWS S3 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchBucket</Code>
  <Message>The specified bucket does not exist</Message>
</Error>`))
	}))
	defer server.Close()

	checker := &NetworkChecker{
		Timeout: 5 * time.Second,
	}

	host := server.URL[7:]

	result := checker.checkHTTPFingerprints(context.Background(), host, "bucket.s3.amazonaws.com", "AWS S3")

	if !result.Vulnerable {
		t.Error("Expected vulnerable=true for AWS S3 fingerprint")
	}

	if result.Provider != "AWS S3" {
		t.Errorf("Expected provider=AWS S3, got %q", result.Provider)
	}
}

func TestCheckHTTPFingerprints_NoVulnerability(t *testing.T) {
	// Create test server that returns normal content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>Normal website content</body></html>"))
	}))
	defer server.Close()

	checker := &NetworkChecker{
		Timeout: 5 * time.Second,
	}

	host := server.URL[7:]

	result := checker.checkHTTPFingerprints(context.Background(), host, "example.com", "Unknown")

	if result.Vulnerable {
		t.Error("Expected vulnerable=false for normal content")
	}
}

func TestCheckSubdomainTakeover_NoCNAME(t *testing.T) {
	checker := &NetworkChecker{
		Timeout: 5 * time.Second,
	}

	// Use a domain that doesn't have a CNAME (e.g., example.com)
	result := checker.checkSubdomainTakeover(context.Background(), "example.com")

	if result.Vulnerable {
		t.Error("Expected vulnerable=false when no CNAME exists")
	}

	if result.CNAME != "" && result.CNAME != "example.com" {
		t.Logf("CNAME: %q (expected empty or same as host)", result.CNAME)
	}
}

func TestAnalyzePortRisks(t *testing.T) {
	checker := &NetworkChecker{}
	netSec := &NetworkSecurityResult{
		OpenPorts: []PortInfo{
			{Port: 23, Service: "telnet", Risk: "critical"},
			{Port: 3389, Service: "rdp", Risk: "critical"},
			{Port: 22, Service: "ssh", Risk: "high"},
			{Port: 3306, Service: "mysql", Risk: "high"},
			{Port: 8080, Service: "http-alt", Risk: "medium"},
			{Port: 80, Service: "http", Risk: "low"},
		},
		Issues: []string{},
		Recommendations: []string{},
	}

	checker.analyzePortRisks(netSec)

	// Should have added descriptions to all ports
	for _, port := range netSec.OpenPorts {
		if port.Description == "" {
			t.Errorf("Port %d should have a description", port.Port)
		}
	}

	// Should have identified critical ports
	hasCriticalIssue := false
	for _, issue := range netSec.Issues {
		if issue == "2 critical port(s) exposed (Telnet/RDP/VNC)" {
			hasCriticalIssue = true
			break
		}
	}
	if !hasCriticalIssue {
		t.Error("Should have identified critical ports issue")
	}

	// Should have identified high-risk ports
	hasHighRiskIssue := false
	for _, issue := range netSec.Issues {
		if issue == "2 high-risk port(s) exposed (SSH/Database/SMB)" {
			hasHighRiskIssue = true
			break
		}
	}
	if !hasHighRiskIssue {
		t.Error("Should have identified high-risk ports issue")
	}

	// Should have recommendations
	if len(netSec.Recommendations) == 0 {
		t.Error("Should have added recommendations")
	}
}

func TestCheckPort_Closed(t *testing.T) {
	checker := &NetworkChecker{
		PortScanTimeout: 1 * time.Second,
	}

	// Try to connect to a port that's likely closed
	result := checker.checkPort(context.Background(), "127.0.0.1", 54321)

	if result != nil {
		t.Error("Expected nil result for closed port")
	}
}

func TestCheckPort_Open(t *testing.T) {
	// Create a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_, _ = conn.Write([]byte("TEST BANNER\n"))
			conn.Close()
		}
	}()

	checker := &NetworkChecker{
		PortScanTimeout: 2 * time.Second,
	}

	result := checker.checkPort(context.Background(), "127.0.0.1", port)

	if result == nil {
		t.Fatal("Expected non-nil result for open port")
	}

	if result.Port != port {
		t.Errorf("Expected port %d, got %d", port, result.Port)
	}

	if result.State != "open" {
		t.Errorf("Expected state=open, got %q", result.State)
	}

	if result.Protocol != "tcp" {
		t.Errorf("Expected protocol=tcp, got %q", result.Protocol)
	}

	// Banner might be captured
	if result.Banner != "" {
		t.Logf("Banner captured: %q", result.Banner)
	}
}

func TestScanPorts(t *testing.T) {
	// Create multiple test TCP servers
	listener1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test server 1: %v", err)
	}
	defer listener1.Close()
	port1 := listener1.Addr().(*net.TCPAddr).Port

	listener2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test server 2: %v", err)
	}
	defer listener2.Close()
	port2 := listener2.Addr().(*net.TCPAddr).Port

	// Accept connections
	go func() {
		for {
			conn, err := listener1.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	go func() {
		for {
			conn, err := listener2.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	checker := &NetworkChecker{
		PortScanTimeout: 2 * time.Second,
		CommonPorts:     []int{port1, port2, 54321}, // Mix of open and closed
		MaxPortWorkers:  3,
	}

	results := checker.scanPorts(context.Background(), "127.0.0.1")

	// Should find exactly 2 open ports
	if len(results) != 2 {
		t.Errorf("Expected 2 open ports, found %d", len(results))
	}

	// Verify the open ports match
	foundPorts := make(map[int]bool)
	for _, result := range results {
		foundPorts[result.Port] = true
	}

	if !foundPorts[port1] {
		t.Errorf("Expected to find port %d in results", port1)
	}

	if !foundPorts[port2] {
		t.Errorf("Expected to find port %d in results", port2)
	}
}

func TestNetworkChecker_Check_Integration(t *testing.T) {
	// Skip in short mode or if network is unavailable
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := &NetworkChecker{
		Timeout:         5 * time.Second,
		PortScanTimeout: 2 * time.Second,
		EnablePortScan:  false, // Disable port scan for faster test
		CommonPorts:     []int{80, 443},
		MaxPortWorkers:  2,
	}

	// Test with a real domain that has a CNAME (subdomain takeover check)
	// Using example.com which should resolve fine (no vulnerability)
	result := checker.Check(context.Background(), "example.com")

	if result.Status != "ok" {
		t.Errorf("Expected status=ok, got %q (error: %s)", result.Status, result.Error)
	}

	if result.NetworkSecurity == nil {
		t.Fatal("Expected NetworkSecurity to be populated")
	}

	if result.NetworkSecurity.SubdomainTakeover == nil {
		t.Error("Expected SubdomainTakeover check to be performed")
	}

	// example.com should not be vulnerable
	if result.NetworkSecurity.SubdomainTakeover.Vulnerable {
		t.Error("example.com should not be vulnerable to subdomain takeover")
	}

	t.Logf("Check result: %+v", result)
	t.Logf("Network Security: %+v", result.NetworkSecurity)
}

func TestNetworkChecker_Check_WithPortScan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping port scan test in short mode")
	}

	// Create a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	checker := &NetworkChecker{
		Timeout:         5 * time.Second,
		PortScanTimeout: 2 * time.Second,
		EnablePortScan:  true,
		CommonPorts:     []int{port, 54321}, // One open, one closed
		MaxPortWorkers:  2,
	}

	result := checker.Check(context.Background(), "127.0.0.1")

	if result.Status != "ok" {
		t.Errorf("Expected status=ok, got %q (error: %s)", result.Status, result.Error)
	}

	if result.NetworkSecurity == nil {
		t.Fatal("Expected NetworkSecurity to be populated")
	}

	if len(result.NetworkSecurity.OpenPorts) != 1 {
		t.Errorf("Expected 1 open port, found %d", len(result.NetworkSecurity.OpenPorts))
	}

	if result.NetworkSecurity.PortScanDuration == 0 {
		t.Error("Expected PortScanDuration to be set")
	}

	t.Logf("Port scan duration: %.2f ms", result.NetworkSecurity.PortScanDuration)
}

func TestNetworkChecker_Check_ContextTimeout(t *testing.T) {
	checker := &NetworkChecker{
		Timeout:         5 * time.Second,
		PortScanTimeout: 2 * time.Second,
		EnablePortScan:  false,
	}

	// Create context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure context is expired

	result := checker.Check(ctx, "example.com")

	// Should complete but might have errors due to context cancellation
	t.Logf("Result with cancelled context: status=%s, error=%s", result.Status, result.Error)
}
