package compliance

// Framework represents a compliance or regulatory framework
type Framework struct {
	ID          string   // Unique identifier (e.g., "iso27001", "pdpa")
	Name        string   // Display name (e.g., "ISO/IEC 27001:2022")
	Description string   // Brief description
	Region      string   // Geographic region (e.g., "Global", "Singapore", "Japan", "Korea")
	Categories  []string // Major categories within the framework
}

// Requirement represents a specific requirement within a compliance framework
type Requirement struct {
	ID          string   // Requirement identifier (e.g., "A.8.24", "Annex A 8.1")
	Title       string   // Requirement title
	Description string   // Detailed description
	Controls    []string // Security check names that satisfy this requirement
}

// ComplianceMapping maps security checks to framework requirements
type ComplianceMapping struct {
	CheckName    string            // Security check name
	Frameworks   map[string][]string // Framework ID -> Requirement IDs
	Priority     map[string]string // Framework ID -> Priority (Critical, High, Medium, Low)
	Notes        map[string]string // Framework ID -> Additional notes
}

// SupportedFrameworks returns all compliance frameworks supported by the tool
func SupportedFrameworks() []Framework {
	return []Framework{
		// Global/International Standards
		{
			ID:          "iso27001",
			Name:        "ISO/IEC 27001:2022",
			Description: "Information Security Management System standard",
			Region:      "Global",
			Categories:  []string{"Organizational", "People", "Physical", "Technological"},
		},
		{
			ID:          "iso27701",
			Name:        "ISO/IEC 27701:2019",
			Description: "Privacy Information Management System extension to ISO 27001",
			Region:      "Global",
			Categories:  []string{"Privacy Controls", "PII Processing", "Data Protection"},
		},

		// Japan
		{
			ID:          "jisq27001",
			Name:        "JIS Q 27001",
			Description: "Japanese Industrial Standard for Information Security Management (aligned with ISO 27001)",
			Region:      "Japan",
			Categories:  []string{"Information Security Management", "Risk Management", "Asset Management"},
		},
		{
			ID:          "jisq27002",
			Name:        "JIS Q 27002",
			Description: "Code of practice for information security controls",
			Region:      "Japan",
			Categories:  []string{"Security Controls", "Implementation Guidance"},
		},
		{
			ID:          "privacymark",
			Name:        "PrivacyMark (Pマーク)",
			Description: "Japanese privacy certification based on JIS Q 15001",
			Region:      "Japan",
			Categories:  []string{"Personal Information Protection", "Privacy Management"},
		},
		{
			ID:          "fisc",
			Name:        "FISC Security Guidelines",
			Description: "Center for Financial Industry Information Systems security standards",
			Region:      "Japan",
			Categories:  []string{"Financial System Security", "Network Security", "Access Control"},
		},

		// Singapore
		{
			ID:          "pdpa",
			Name:        "PDPA (Personal Data Protection Act)",
			Description: "Singapore's personal data protection law",
			Region:      "Singapore",
			Categories:  []string{"Consent", "Data Protection", "Accountability"},
		},
		{
			ID:          "mtcs",
			Name:        "MTCS SS 584",
			Description: "Multi-Tier Cloud Security Singapore Standard",
			Region:      "Singapore",
			Categories:  []string{"Cloud Security", "Data Security", "Infrastructure Security"},
		},

		// South Korea
		{
			ID:          "kisms",
			Name:        "K-ISMS (Korea ISMS)",
			Description: "Korean Information Security Management System certification",
			Region:      "Korea",
			Categories:  []string{"Management Process", "Protection Measures", "Information Security"},
		},
		{
			ID:          "ismsp",
			Name:        "ISMS-P",
			Description: "Integrated certification for ISMS and Personal Information Protection",
			Region:      "Korea",
			Categories:  []string{"Information Security", "Personal Information Protection"},
		},
		{
			ID:          "pims",
			Name:        "PIMS (Personal Information Management System)",
			Description: "Korean Personal Information Protection certification",
			Region:      "Korea",
			Categories:  []string{"Privacy Protection", "Personal Data Management"},
		},
		{
			ID:          "pipl",
			Name:        "PIPL (Personal Information Protection Law)",
			Description: "Korean personal information protection regulation",
			Region:      "Korea",
			Categories:  []string{"Data Rights", "Security Measures", "Breach Notification"},
		},
	}
}

// GetFramework returns a specific framework by ID
func GetFramework(id string) *Framework {
	for _, fw := range SupportedFrameworks() {
		if fw.ID == id {
			return &fw
		}
	}
	return nil
}

// GetFrameworksByRegion returns all frameworks for a specific region
func GetFrameworksByRegion(region string) []Framework {
	var frameworks []Framework
	for _, fw := range SupportedFrameworks() {
		if fw.Region == region || fw.Region == "Global" {
			frameworks = append(frameworks, fw)
		}
	}
	return frameworks
}
