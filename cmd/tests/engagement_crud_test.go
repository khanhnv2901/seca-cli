package tests

import (
	"testing"
	"time"

	cmdpkg "github.com/khanhnv2901/seca-cli/cmd"
	"github.com/khanhnv2901/seca-cli/cmd/testutil"
	_ "unsafe"
)

//go:linkname linkedLoadEngagements github.com/khanhnv2901/seca-cli/cmd.loadEngagements
func linkedLoadEngagements() []cmdpkg.Engagement

//go:linkname linkedSaveEngagements github.com/khanhnv2901/seca-cli/cmd.saveEngagements
func linkedSaveEngagements(list []cmdpkg.Engagement)

//go:linkname linkedAddScopeEntries github.com/khanhnv2901/seca-cli/cmd.addScopeEntries
func linkedAddScopeEntries(id string, entries []string) error

//go:linkname linkedGetEngagementsFilePath github.com/khanhnv2901/seca-cli/cmd.getEngagementsFilePath
func linkedGetEngagementsFilePath() (string, error)

func TestEngagementCRUD(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	cleanup := testutil.SetupEngagementsFile(t, linkedGetEngagementsFilePath)
	defer cleanup()

	created := cmdpkg.Engagement{
		ID:        "crud-1",
		Name:      "CRUD Test",
		Owner:     "owner@example.com",
		CreatedAt: time.Now(),
		ROEAgree:  true,
	}

	linkedSaveEngagements([]cmdpkg.Engagement{created})

	engagements := linkedLoadEngagements()
	if len(engagements) != 1 {
		t.Fatalf("expected 1 engagement after create, got %d", len(engagements))
	}
	if engagements[0].Name != created.Name {
		t.Fatalf("expected name %s, got %s", created.Name, engagements[0].Name)
	}

	if err := linkedAddScopeEntries(created.ID, []string{"https://example.com"}); err != nil {
		t.Fatalf("add scope failed: %v", err)
	}

	updated := linkedLoadEngagements()
	if len(updated[0].Scope) != 1 || updated[0].Scope[0] != "https://example.com" {
		t.Fatalf("expected updated scope entry, got %+v", updated[0].Scope)
	}

	pruned := make([]cmdpkg.Engagement, 0)
	for _, eng := range updated {
		if eng.ID != created.ID {
			pruned = append(pruned, eng)
		}
	}
	linkedSaveEngagements(pruned)

	final := linkedLoadEngagements()
	if len(final) != 0 {
		t.Fatalf("expected no engagements after delete simulation, got %d", len(final))
	}
}
