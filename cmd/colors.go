package cmd

import (
	"strings"

	"github.com/fatih/color"
)

var (
	colorSuccess = color.New(color.FgGreen).SprintFunc()
	colorInfo    = color.New(color.FgCyan).SprintFunc()
	colorWarn    = color.New(color.FgYellow).SprintFunc()
	colorError   = color.New(color.FgRed).SprintFunc()
)

func formatStatusWithColor(status string) string {
	switch strings.ToLower(status) {
	case "ok", "success", "pass":
		return colorSuccess(status)
	case "error", "fail", "failed":
		return colorError(status)
	default:
		return status
	}
}
