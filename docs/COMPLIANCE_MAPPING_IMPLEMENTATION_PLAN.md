# Compliance Mapping Engine - Implementation Plan

## Overview
This document outlines the implementation plan for adding a compliance mapping engine to SECA-CLI. This feature will allow users to filter and view security scan results based on their relevant compliance framework (ISO 27001, PDPA, K-ISMS, etc.).

## Background
Clients from different regions (Japan, Singapore, South Korea) are only familiar with their country's compliance frameworks. We need to provide an engine that helps them check and display security information according to the compliance framework they want.

## Current Status

### Completed ✅
1. **Compliance Framework Definitions** - Created `/home/khanhnv/Projects/seca-cli/internal/compliance/framework.go`
   - Defined 12 compliance frameworks:
     - Global: ISO/IEC 27001:2022, ISO/IEC 27701:2019
     - Japan: JIS Q 27001, JIS Q 27002, PrivacyMark, FISC Security Guidelines
     - Singapore: PDPA, MTCS SS 584
     - South Korea: K-ISMS, ISMS-P, PIMS, PIPL
   - Created `Framework` struct with ID, Name, Description, Region, Categories
   - Implemented helper functions: `SupportedFrameworks()`, `GetFramework()`, `GetFrameworksByRegion()`

2. **Compliance Mappings** - Created `/home/khanhnv/Projects/seca-cli/internal/compliance/mappings.go`
   - Mapped all 36 security checks to compliance requirements
   - Each mapping includes:
     - Framework ID → Requirement IDs (e.g., ISO 27001: A.8.24)
     - Priority per framework (Critical, High, Medium, Low)
     - Extensible structure for future notes
   - Implemented helper functions: `GetMappingForCheck()`, `GetChecksForFramework()`, `GetRequirementsForFramework()`

3. **Vulnerability Structure Extension** - Modified `/home/khanhnv/Projects/seca-cli/internal/checker/vulnerability.go`
   - Added `ComplianceMapping` field to `Vulnerability` struct
   - Created `ComplianceDetails` struct (Requirements, Priority, Notes)
   - Implemented `EnrichWithComplianceData()` function
   - Implemented `FilterByCompliance()` function

### Pending ⏳
4. Add CLI flag for compliance framework selection
5. Update report generation logic for compliance filtering
6. Modify HTML template for compliance-specific display
7. Add compliance framework listing command
8. Update documentation

## Implementation Steps

### Step 1: Add CLI Flags for Compliance Framework Selection
**File:** `/home/khanhnv/Projects/seca-cli/cmd/check.go`

**Tasks:**
- [ ] Add `--compliance` flag to the `check` command
- [ ] Add `--list-frameworks` flag to list available frameworks
- [ ] Add `--list-frameworks-by-region` flag to filter by region
- [ ] Validate framework ID against supported frameworks
- [ ] Store selected framework in context/config

**Example CLI usage:**
```bash
# List all available compliance frameworks
./seca check --list-frameworks

# List frameworks for a specific region
./seca check --list-frameworks-by-region Singapore

# Run scan with compliance filter
./seca check --url https://example.com --compliance iso27001

# Run scan with multiple frameworks
./seca check --url https://example.com --compliance iso27001,pdpa
```

**Code changes needed:**
```go
// In cmd/check.go
var (
    complianceFramework string
    listFrameworks      bool
    frameworkRegion     string
)

func init() {
    checkCmd.Flags().StringVar(&complianceFramework, "compliance", "", "Filter results by compliance framework (e.g., iso27001, pdpa, kisms)")
    checkCmd.Flags().BoolVar(&listFrameworks, "list-frameworks", false, "List all available compliance frameworks")
    checkCmd.Flags().StringVar(&frameworkRegion, "list-frameworks-by-region", "", "List frameworks for a specific region (Global, Japan, Singapore, Korea)")
}
```

### Step 2: Integrate Compliance Mapping into Report Generation
**File:** `/home/khanhnv/Projects/seca-cli/cmd/report.go`

**Tasks:**
- [ ] Import the `internal/compliance` package
- [ ] Modify `generateReport()` to accept compliance framework parameter
- [ ] Call `EnrichWithComplianceData()` before generating report
- [ ] If framework specified, filter vulnerabilities using `FilterByCompliance()`
- [ ] Pass compliance metadata to HTML template

**Code changes needed:**
```go
// In cmd/report.go
import (
    "github.com/yourusername/seca-cli/internal/compliance"
)

func generateReport(vulnReport *checker.VulnerabilityReport, frameworkID string) error {
    // Enrich vulnerabilities with compliance data
    mappingsProvider := func(checkName string) map[string][]string {
        mapping := compliance.GetMappingForCheck(checkName)
        if mapping != nil {
            return mapping.Frameworks
        }
        return nil
    }

    priorityProvider := func(checkName string) map[string]string {
        mapping := compliance.GetMappingForCheck(checkName)
        if mapping != nil {
            return mapping.Priority
        }
        return nil
    }

    vulnReport.Vulnerabilities = checker.EnrichWithComplianceData(
        vulnReport.Vulnerabilities,
        mappingsProvider,
        priorityProvider,
    )

    // Filter by compliance framework if specified
    if frameworkID != "" {
        vulnReport.Vulnerabilities = checker.FilterByCompliance(
            vulnReport.Vulnerabilities,
            frameworkID,
        )
    }

    // Generate report...
}
```

### Step 3: Extend Vulnerability Report Structure
**File:** `/home/khanhnv/Projects/seca-cli/internal/checker/vulnerability.go`

**Tasks:**
- [ ] Add `ComplianceFramework` field to `VulnerabilityReport` struct
- [ ] Add `ComplianceSummary` struct to show framework-specific summary
- [ ] Update JSON output to include compliance information

**Code changes needed:**
```go
type VulnerabilityReport struct {
    ScanDate              string               `json:"scan_date"`
    Duration              string               `json:"duration"`
    ScanURL               string               `json:"scan_url"`
    Status                string               `json:"status"`
    TotalURLsScanned      int                  `json:"total_urls_scanned"`
    Vulnerabilities       []Vulnerability      `json:"vulnerabilities"`
    Summary               VulnerabilitySummary `json:"summary"`
    ResultSources         []string             `json:"result_sources,omitempty"`
    ComplianceFramework   *Framework           `json:"compliance_framework,omitempty"`   // NEW
    ComplianceSummary     *ComplianceSummary   `json:"compliance_summary,omitempty"`     // NEW
}

type ComplianceSummary struct {
    FrameworkID          string            `json:"framework_id"`
    FrameworkName        string            `json:"framework_name"`
    TotalChecks          int               `json:"total_checks"`
    PassedChecks         int               `json:"passed_checks"`
    FailedChecks         int               `json:"failed_checks"`
    ComplianceScore      float64           `json:"compliance_score"` // Percentage
    RequirementsCovered  map[string]string `json:"requirements_covered"` // Requirement ID -> Status
}
```

### Step 4: Update HTML Template for Compliance Display
**File:** `/home/khanhnv/Projects/seca-cli/cmd/templates/report.html`

**Tasks:**
- [ ] Add compliance framework header section (if framework selected)
- [ ] Display compliance-specific information for each vulnerability
- [ ] Show requirement IDs and priority badges
- [ ] Add compliance summary dashboard
- [ ] Color-code checks by compliance priority
- [ ] Add filter/toggle to show only compliance-relevant checks

**Template additions needed:**
```html
<!-- Compliance Framework Header -->
{{if .ComplianceFramework}}
<div class="compliance-header">
    <h2>Compliance Framework: {{.ComplianceFramework.Name}}</h2>
    <div class="compliance-info">
        <span>Region: {{.ComplianceFramework.Region}}</span>
        <span>Total Checks: {{.ComplianceSummary.TotalChecks}}</span>
        <span>Compliance Score: {{.ComplianceSummary.ComplianceScore}}%</span>
    </div>
</div>
{{end}}

<!-- In vulnerability details -->
{{if .ComplianceMapping}}
<div class="compliance-details">
    <h4>Compliance Requirements</h4>
    {{range $framework, $details := .ComplianceMapping}}
    <div class="framework-mapping">
        <strong>{{$framework}}:</strong>
        <span class="priority-badge priority-{{$details.Priority | lower}}">
            {{$details.Priority}}
        </span>
        <ul>
            {{range $details.Requirements}}
            <li>{{.}}</li>
            {{end}}
        </ul>
    </div>
    {{end}}
</div>
{{end}}
```

**CSS additions:**
```css
.compliance-header {
    background: #f8f9fa;
    padding: 20px;
    margin-bottom: 20px;
    border-radius: 8px;
    border-left: 4px solid #007bff;
}

.priority-badge {
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: bold;
}

.priority-critical {
    background: #dc3545;
    color: white;
}

.priority-high {
    background: #fd7e14;
    color: white;
}

.priority-medium {
    background: #ffc107;
    color: #000;
}

.priority-low {
    background: #28a745;
    color: white;
}
```

### Step 5: Add Compliance Framework Listing Command
**File:** `/home/khanhnv/Projects/seca-cli/cmd/compliance.go` (NEW)

**Tasks:**
- [ ] Create new `compliance` subcommand
- [ ] Implement `list` action to show all frameworks
- [ ] Implement `info <framework-id>` to show framework details
- [ ] Implement `checks <framework-id>` to show all checks for a framework
- [ ] Format output as a table

**Example usage:**
```bash
# List all frameworks
./seca compliance list

# Show framework info
./seca compliance info iso27001

# List all security checks for a framework
./seca compliance checks pdpa
```

**Implementation:**
```go
package cmd

import (
    "fmt"
    "os"
    "text/tabwriter"

    "github.com/spf13/cobra"
    "github.com/yourusername/seca-cli/internal/compliance"
)

var complianceCmd = &cobra.Command{
    Use:   "compliance",
    Short: "Manage compliance frameworks",
    Long:  "List, view, and manage compliance framework mappings",
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List all available compliance frameworks",
    Run: func(cmd *cobra.Command, args []string) {
        frameworks := compliance.SupportedFrameworks()

        w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
        fmt.Fprintln(w, "ID\tNAME\tREGION\tDESCRIPTION")
        fmt.Fprintln(w, "---\t----\t------\t-----------")

        for _, fw := range frameworks {
            fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", fw.ID, fw.Name, fw.Region, fw.Description)
        }

        w.Flush()
    },
}

func init() {
    rootCmd.AddCommand(complianceCmd)
    complianceCmd.AddCommand(listCmd)
}
```

### Step 6: Update JSON Output Format
**File:** `/home/khanhnv/Projects/seca-cli/cmd/report.go`

**Tasks:**
- [ ] Ensure compliance data is included in JSON export
- [ ] Add `--export-compliance` flag for detailed compliance report
- [ ] Generate separate compliance summary JSON

**Example JSON structure:**
```json
{
  "scan_date": "2025-11-14T02:23:46+07:00",
  "compliance_framework": {
    "id": "iso27001",
    "name": "ISO/IEC 27001:2022",
    "region": "Global"
  },
  "compliance_summary": {
    "framework_id": "iso27001",
    "total_checks": 28,
    "passed_checks": 18,
    "failed_checks": 10,
    "compliance_score": 64.3,
    "requirements_covered": {
      "A.8.24": "Passed",
      "A.8.16": "Failed"
    }
  },
  "vulnerabilities": [
    {
      "name": "HTTPS enabled",
      "severity": "Critical",
      "status": "Passed",
      "compliance_mapping": {
        "iso27001": {
          "requirements": ["A.8.24", "A.8.9"],
          "priority": "Critical"
        }
      }
    }
  ]
}
```

### Step 7: Add Configuration File Support
**File:** `/home/khanhnv/Projects/seca-cli/cmd/config.go`

**Tasks:**
- [ ] Allow users to set default compliance framework in config
- [ ] Support `.seca.yaml` or `.seca.json` in project root
- [ ] Add `default_framework` configuration option

**Example config:**
```yaml
# .seca.yaml
default_framework: iso27001
frameworks:
  - iso27001
  - pdpa
```

### Step 8: Documentation Updates

**Tasks:**
- [ ] Update main README.md with compliance feature
- [ ] Create `/home/khanhnv/Projects/seca-cli/docs/compliance-frameworks.md`
- [ ] Document all supported frameworks and their mappings
- [ ] Add examples of compliance-filtered scans
- [ ] Create compliance mapping table showing which checks map to which requirements

**Documentation structure:**
```
docs/
├── compliance-frameworks.md          # List all frameworks
├── compliance-mappings.md            # Security check to requirement mappings
├── usage-compliance.md               # Usage examples
└── materials/
    └── compliance-mapping-table.md   # Detailed mapping table
```

### Step 9: Testing

**Tasks:**
- [ ] Create unit tests for compliance filtering
- [ ] Test each framework filter
- [ ] Test HTML report rendering with compliance data
- [ ] Test JSON export with compliance data
- [ ] Integration test for full compliance workflow
- [ ] Test edge cases (no mappings, invalid framework ID)

**Test files to create:**
```
internal/compliance/framework_test.go
internal/compliance/mappings_test.go
cmd/compliance_test.go
```

### Step 10: Additional Enhancements (Future)

**Optional features for later:**
- [ ] Compliance gap analysis (what's missing to be compliant)
- [ ] Compliance trend tracking over time
- [ ] Export compliance audit report (PDF)
- [ ] Integration with compliance management platforms
- [ ] Custom compliance framework definitions
- [ ] Compliance checklist generator
- [ ] Automated remediation suggestions per framework

## File Structure After Implementation

```
/home/khanhnv/Projects/seca-cli/
├── cmd/
│   ├── check.go                    # Modified: Add --compliance flag
│   ├── compliance.go               # NEW: Compliance subcommand
│   ├── report.go                   # Modified: Integrate compliance
│   └── templates/
│       └── report.html             # Modified: Compliance display
├── internal/
│   ├── checker/
│   │   └── vulnerability.go        # Modified: ComplianceMapping field
│   └── compliance/
│       ├── framework.go            # ✅ Created
│       ├── mappings.go             # ✅ Created
│       ├── framework_test.go       # NEW: Tests
│       └── mappings_test.go        # NEW: Tests
└── docs/
    ├── COMPLIANCE_MAPPING_IMPLEMENTATION_PLAN.md  # This file
    ├── compliance-frameworks.md    # NEW: Framework docs
    ├── compliance-mappings.md      # NEW: Mapping docs
    └── usage-compliance.md         # NEW: Usage examples
```

## Migration Strategy

1. **Backward Compatibility:** All compliance features are opt-in via flags. Existing functionality remains unchanged.
2. **Default Behavior:** Without `--compliance` flag, all checks are shown (current behavior)
3. **Data Format:** JSON output includes compliance data but old parsers ignore it
4. **Versioning:** Consider this a minor version bump (e.g., v1.3.0 → v1.4.0)

## Testing Checklist

- [ ] Test without compliance flag (backward compatibility)
- [ ] Test with valid framework ID
- [ ] Test with invalid framework ID (should show error + list available frameworks)
- [ ] Test with multiple frameworks
- [ ] Test framework listing commands
- [ ] Test HTML report rendering with compliance data
- [ ] Test JSON export with compliance data
- [ ] Test each supported framework filter
- [ ] Test region-based framework listing
- [ ] Verify all 36 security checks have mappings

## Next Session Tasks (Priority Order)

1. **Implement CLI flags** (Step 1) - 30 minutes
2. **Integrate compliance into report generation** (Step 2) - 45 minutes
3. **Update HTML template** (Step 4) - 1 hour
4. **Add compliance command** (Step 5) - 30 minutes
5. **Test and validate** (Step 9) - 30 minutes

## Notes

- **Current State:** Framework definitions and mappings are complete. Vulnerability structure is extended.
- **Remaining Work:** CLI integration, report generation updates, HTML template updates
- **Estimated Time:** ~3-4 hours for core functionality
- **Dependencies:** None - all code is self-contained

## References

- ISO/IEC 27001:2022 Annex A controls
- PDPA Guidelines (Singapore)
- K-ISMS certification requirements
- MTCS SS 584 standard
- FISC Security Guidelines
- PrivacyMark (JIS Q 15001)

---

**Document Version:** 1.0
**Created:** 2025-11-14
**Last Updated:** 2025-11-14
**Status:** Ready for implementation
