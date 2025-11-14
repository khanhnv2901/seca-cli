package cmd

import (
	"testing"

	"github.com/fatih/color"
)

func TestFormatStatusWithColor(t *testing.T) {
	original := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = original
	})

	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "success", status: "OK", want: "OK"},
		{name: "pass synonym", status: "pass", want: "pass"},
		{name: "failure", status: "FAILED", want: "FAILED"},
		{name: "unknown", status: "pending", want: "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatStatusWithColor(tt.status); got != tt.want {
				t.Fatalf("formatStatusWithColor(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
