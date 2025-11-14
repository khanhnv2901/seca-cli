package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestProgressPrinterLifecycle(t *testing.T) {
	printer := newProgressPrinter(0, "HTTP")
	if printer.total != 1 {
		t.Fatalf("expected total to be clamped to 1, got %d", printer.total)
	}

	output := captureStdout(t, func() {
		printer.Start()
		printer.Increment(true, 0.5)
		printer.Increment(false, 1.0)
		time.Sleep(350 * time.Millisecond) // allow ticker to tick at least once
		printer.Stop()
		time.Sleep(50 * time.Millisecond) // ensure loop goroutine exits
	})

	if !strings.Contains(output, "Progress: 2/2") {
		t.Fatalf("expected summary progress, got %q", output)
	}
	if !strings.Contains(output, "OK:1") || !strings.Contains(output, "Fail:1") {
		t.Fatalf("expected OK/Fail counts in output, got %q", output)
	}
	if !strings.Contains(output, "Avg:0.75s") {
		t.Fatalf("expected average duration in output, got %q", output)
	}
}
