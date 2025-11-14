package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
)

func main() {
	// Check if target is provided
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run network_security_example.go <target>")
		fmt.Println("Example: go run network_security_example.go example.com")
		os.Exit(1)
	}

	target := os.Args[1]

	fmt.Printf("🔍 Running Network Security Check on: %s\n\n", target)

	// Create network checker with configuration
	netChecker := &checker.NetworkChecker{
		Timeout:         10 * time.Second,
		PortScanTimeout: 2 * time.Second,
		EnablePortScan:  true,
		// Scan common web and database ports
		CommonPorts:    []int{21, 22, 23, 80, 443, 3306, 3389, 5432, 5900, 6379, 8080, 8443, 27017},
		MaxPortWorkers: 10,
	}

	// Run the check
	fmt.Println("⏳ Scanning... (this may take 5-10 seconds)")
	result := netChecker.Check(context.Background(), target)

	// Display results
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("NETWORK SECURITY SCAN RESULTS")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	fmt.Printf("Target: %s\n", result.Target)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Checked At: %s\n\n", result.CheckedAt.Format(time.RFC3339))

	if result.Error != "" {
		fmt.Printf("❌ Error: %s\n", result.Error)
		os.Exit(1)
	}

	if result.NetworkSecurity == nil {
		fmt.Println("⚠️  No network security data available")
		os.Exit(1)
	}

	netSec := result.NetworkSecurity

	// 1. Display Subdomain Takeover Results
	fmt.Println("📡 SUBDOMAIN TAKEOVER CHECK")
	fmt.Println(strings.Repeat("-", 60))
	if netSec.SubdomainTakeover != nil {
		takeover := netSec.SubdomainTakeover

		if takeover.Vulnerable {
			fmt.Printf("⚠️  VULNERABLE TO SUBDOMAIN TAKEOVER!\n\n")
			fmt.Printf("  CNAME: %s\n", takeover.CNAME)
			fmt.Printf("  Provider: %s\n", takeover.Provider)
			fmt.Printf("  Confidence: %s\n", takeover.Confidence)
			fmt.Printf("  Fingerprint: %s\n", takeover.Fingerprint)

			if takeover.HTTPStatusCode > 0 {
				fmt.Printf("  HTTP Status: %d\n", takeover.HTTPStatusCode)
			}

			if takeover.Recommendation != "" {
				fmt.Printf("\n  💡 Recommendation:\n")
				fmt.Printf("  %s\n", takeover.Recommendation)
			}
		} else {
			fmt.Println("✅ No subdomain takeover vulnerability detected")

			if takeover.CNAME != "" && takeover.CNAME != target {
				fmt.Printf("  CNAME: %s → %s (resolves correctly)\n", target, takeover.CNAME)
			} else {
				fmt.Println("  No CNAME record found")
			}
		}
	} else {
		fmt.Println("⚠️  Subdomain takeover check not performed")
	}

	// 2. Display Open Ports Results
	fmt.Println("\n🔓 OPEN PORTS SCAN")
	fmt.Println(strings.Repeat("-", 60))

	if len(netSec.OpenPorts) == 0 {
		fmt.Println("✅ No open ports found (or port scanning disabled)")
	} else {
		fmt.Printf("Found %d open port(s):\n\n", len(netSec.OpenPorts))

		// Group by risk level
		critical := []checker.PortInfo{}
		high := []checker.PortInfo{}
		medium := []checker.PortInfo{}
		low := []checker.PortInfo{}
		info := []checker.PortInfo{}

		for _, port := range netSec.OpenPorts {
			switch port.Risk {
			case "critical":
				critical = append(critical, port)
			case "high":
				high = append(high, port)
			case "medium":
				medium = append(medium, port)
			case "low":
				low = append(low, port)
			default:
				info = append(info, port)
			}
		}

		// Display by risk level
		if len(critical) > 0 {
			fmt.Println("  🔴 CRITICAL RISK:")
			for _, port := range critical {
				displayPort(port)
			}
		}

		if len(high) > 0 {
			fmt.Println("\n  🟠 HIGH RISK:")
			for _, port := range high {
				displayPort(port)
			}
		}

		if len(medium) > 0 {
			fmt.Println("\n  🟡 MEDIUM RISK:")
			for _, port := range medium {
				displayPort(port)
			}
		}

		if len(low) > 0 {
			fmt.Println("\n  🟢 LOW RISK:")
			for _, port := range low {
				displayPort(port)
			}
		}

		if len(info) > 0 {
			fmt.Println("\n  ℹ️  INFO:")
			for _, port := range info {
				displayPort(port)
			}
		}

		fmt.Printf("\n  ⏱️  Scan Duration: %.2f ms\n", netSec.PortScanDuration)
	}

	// 3. Display Issues and Recommendations
	if len(netSec.Issues) > 0 {
		fmt.Println("\n⚠️  ISSUES IDENTIFIED")
		fmt.Println(strings.Repeat("-", 60))
		for i, issue := range netSec.Issues {
			fmt.Printf("%d. %s\n", i+1, issue)
		}
	}

	if len(netSec.Recommendations) > 0 {
		fmt.Println("\n💡 RECOMMENDATIONS")
		fmt.Println(strings.Repeat("-", 60))
		for i, rec := range netSec.Recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
	}

	// 4. Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("Total Open Ports: %d\n", len(netSec.OpenPorts))
	fmt.Printf("Critical Issues: %d\n", countCriticalPorts(netSec.OpenPorts))
	fmt.Printf("High Risk Issues: %d\n", countHighRiskPorts(netSec.OpenPorts))
	fmt.Printf("Subdomain Takeover: %s\n", boolToStatus(netSec.SubdomainTakeover != nil && netSec.SubdomainTakeover.Vulnerable))

	// Overall risk assessment
	fmt.Println("\n📊 OVERALL RISK:")
	overallRisk := assessOverallRisk(netSec)
	fmt.Printf("  %s\n", overallRisk)

	// 5. Export JSON (optional)
	fmt.Println("\n💾 Exporting detailed results to network_security_results.json...")
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error exporting JSON: %v\n", err)
	} else {
		err = os.WriteFile("network_security_results.json", jsonData, 0644)
		if err != nil {
			fmt.Printf("Error writing file: %v\n", err)
		} else {
			fmt.Println("✅ Results exported successfully!")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Scan complete!")
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

func displayPort(port checker.PortInfo) {
	fmt.Printf("    Port %d (%s) - %s\n", port.Port, port.Service, port.State)
	if port.Description != "" {
		fmt.Printf("      %s\n", port.Description)
	}
	if port.Banner != "" {
		fmt.Printf("      Banner: %s\n", port.Banner)
	}
}

func countCriticalPorts(ports []checker.PortInfo) int {
	count := 0
	for _, port := range ports {
		if port.Risk == "critical" {
			count++
		}
	}
	return count
}

func countHighRiskPorts(ports []checker.PortInfo) int {
	count := 0
	for _, port := range ports {
		if port.Risk == "high" {
			count++
		}
	}
	return count
}

func boolToStatus(b bool) string {
	if b {
		return "⚠️  VULNERABLE"
	}
	return "✅ SAFE"
}

func assessOverallRisk(netSec *checker.NetworkSecurityResult) string {
	criticalCount := countCriticalPorts(netSec.OpenPorts)
	highCount := countHighRiskPorts(netSec.OpenPorts)
	takeover := netSec.SubdomainTakeover != nil && netSec.SubdomainTakeover.Vulnerable

	if criticalCount > 0 || takeover {
		return "🔴 CRITICAL - Immediate action required!"
	}

	if highCount > 2 {
		return "🟠 HIGH - Multiple high-risk ports exposed"
	}

	if highCount > 0 {
		return "🟡 MEDIUM - Some high-risk ports exposed"
	}

	if len(netSec.OpenPorts) > 0 {
		return "🟢 LOW - Only standard ports exposed"
	}

	return "✅ EXCELLENT - No significant issues detected"
}
