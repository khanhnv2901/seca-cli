package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSecurityCheckCatalogMatchesDoc(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	baseDir := filepath.Dir(file)
	projectRoot := filepath.Clean(filepath.Join(baseDir, ".."))
	docPath := filepath.Join(projectRoot, "docs", "materials", "list-of-security-check.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	var docEntries []SecurityCheckSpec
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "|") == false {
			continue
		}
		if strings.Contains(line, "---") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		category := strings.TrimSpace(parts[2])
		if name == "Name" || name == "" {
			continue
		}
		docEntries = append(docEntries, SecurityCheckSpec{Name: name, Category: category})
	}

	catalog := getSecurityCheckCatalog()
	if len(catalog) != len(docEntries) {
		t.Fatalf("catalog length %d does not match doc %d", len(catalog), len(docEntries))
	}

	for i := range catalog {
		if catalog[i] != docEntries[i] {
			t.Fatalf("catalog entry %d = %+v, doc entry = %+v", i, catalog[i], docEntries[i])
		}
	}
}
