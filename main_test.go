package main

import (
	"sync/atomic"
	"testing"

	"github.com/khanhnv2901/seca-cli/cmd"
)

func TestMainInvokesExecute(t *testing.T) {
	var called int32
	execCmd = func() {
		atomic.AddInt32(&called, 1)
	}
	defer func() { execCmd = cmd.Execute }()

	main()

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected execCmd to be invoked once, got %d", called)
	}
}
