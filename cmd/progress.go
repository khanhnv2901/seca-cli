package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type progressPrinter struct {
	total    int
	name     string
	mu       sync.Mutex
	ok       int
	fail     int
	duration float64
	updates  chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func newProgressPrinter(total int, name string) *progressPrinter {
	if total <= 0 {
		total = 1
	}
	return &progressPrinter{
		total:   total,
		name:    name,
		updates: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
}

func (p *progressPrinter) Start() {
	go p.loop()
}

func (p *progressPrinter) Increment(success bool, duration float64) {
	p.mu.Lock()
	if success {
		p.ok++
	} else {
		p.fail++
	}
	p.duration += duration
	p.mu.Unlock()

	select {
	case p.updates <- struct{}{}:
	default:
	}
}

func (p *progressPrinter) Stop() {
	p.stopOnce.Do(func() {
		close(p.done)
	})
	fmt.Fprintf(os.Stdout, "\r%s\r", strings.Repeat(" ", 80))
	p.print()
	fmt.Fprintln(os.Stdout)
}

func (p *progressPrinter) loop() {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.updates:
			p.print()
		case <-ticker.C:
			p.print()
		case <-p.done:
			return
		}
	}
}

func (p *progressPrinter) print() {
	p.mu.Lock()
	ok := p.ok
	fail := p.fail
	dur := p.duration
	p.mu.Unlock()

	completed := ok + fail
	if completed > p.total {
		p.total = completed
	}

	percent := (float64(completed) / float64(p.total)) * 100
	avg := 0.0
	if completed > 0 {
		avg = dur / float64(completed)
	}

	line := fmt.Sprintf("\r[%s] Progress: %d/%d (%.1f%%) OK:%d Fail:%d Avg:%.2fs",
		p.name, completed, p.total, percent, ok, fail, avg)
	fmt.Fprintf(os.Stdout, "%s", line)
}
