package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Job struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	ResultID   string     `json:"result_id,omitempty"`
	Error      string     `json:"error,omitempty"`
}

type JobRequest struct {
	Type         string `json:"type"`
	EngagementID string `json:"engagement_id"`
}

type JobManager struct {
	mu          sync.RWMutex
	jobs        map[string]*Job
	subscribers map[chan Job]struct{}
	maxJobs     int // Maximum number of jobs to keep in memory
}

func NewJobManager() *JobManager {
	m := &JobManager{
		jobs:        make(map[string]*Job),
		subscribers: make(map[chan Job]struct{}),
		maxJobs:     1000, // Default: keep last 1000 jobs
	}
	// Start cleanup goroutine to remove old completed jobs
	go m.cleanupLoop()
	return m
}

func (m *JobManager) CreateJob(jobType, resultID string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := &Job{
		ID:       generateID("job"),
		Type:     jobType,
		Status:   "pending",
		ResultID: resultID,
	}
	m.jobs[job.ID] = job
	m.broadcast(*job)
	return job
}

func (m *JobManager) UpdateJob(id string, update func(*Job)) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil
	}
	update(job)
	m.broadcast(*job)
	return job
}

func (m *JobManager) GetJob(id string) *Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if job, ok := m.jobs[id]; ok {
		copy := *job
		return &copy
	}
	return nil
}

func (m *JobManager) ListJobs(limit int) []Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.jobs) {
		limit = len(m.jobs)
	}
	jobs := make([]Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, *job)
	}

	// Sort newest first by StartedAt using efficient O(n log n) algorithm
	sort.Slice(jobs, func(i, j int) bool {
		// Sort by StartedAt descending (newest first)
		// If StartedAt is nil (not started), sort by ID
		if jobs[i].StartedAt == nil && jobs[j].StartedAt == nil {
			return jobs[i].ID > jobs[j].ID
		}
		if jobs[i].StartedAt == nil {
			return false
		}
		if jobs[j].StartedAt == nil {
			return true
		}
		return jobs[i].StartedAt.After(*jobs[j].StartedAt)
	})

	if limit < len(jobs) {
		jobs = jobs[:limit]
	}
	return jobs
}

func (m *JobManager) Subscribe() (chan Job, func()) {
	// Increased buffer size to reduce risk of dropped updates
	// In production, consider a queue-based approach for slow consumers
	ch := make(chan Job, 10)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch, func() {
		m.mu.Lock()
		if _, ok := m.subscribers[ch]; ok {
			delete(m.subscribers, ch)
			close(ch)
		}
		m.mu.Unlock()
	}
}

func (m *JobManager) broadcast(job Job) {
	for ch := range m.subscribers {
		select {
		case ch <- job:
			// Successfully sent update
		default:
			// Channel buffer full - slow consumer detected
			// TODO: Add logging/metrics to track dropped updates
			// Consider implementing a separate goroutine per subscriber
			// or a persistent queue for production use
		}
	}
}

func generateID(prefix string) string {
	// Use cryptographically secure random ID to prevent enumeration attacks
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

// cleanupLoop removes old completed jobs to prevent unbounded memory growth
func (m *JobManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()

		// If we're under the limit, skip cleanup
		if len(m.jobs) <= m.maxJobs {
			m.mu.Unlock()
			continue
		}

		// Collect completed jobs sorted by finish time
		type jobWithTime struct {
			id   string
			time time.Time
		}
		var completedJobs []jobWithTime

		for id, job := range m.jobs {
			// Only cleanup completed or errored jobs
			if job.Status == "done" || job.Status == "error" {
				finishTime := time.Now()
				if job.FinishedAt != nil {
					finishTime = *job.FinishedAt
				}
				completedJobs = append(completedJobs, jobWithTime{
					id:   id,
					time: finishTime,
				})
			}
		}

		// Sort oldest first
		sort.Slice(completedJobs, func(i, j int) bool {
			return completedJobs[i].time.Before(completedJobs[j].time)
		})

		// Remove oldest jobs until we're under the limit
		toRemove := len(m.jobs) - m.maxJobs
		if toRemove > len(completedJobs) {
			toRemove = len(completedJobs)
		}

		for i := 0; i < toRemove; i++ {
			delete(m.jobs, completedJobs[i].id)
		}

		m.mu.Unlock()
	}
}

// SetMaxJobs configures the maximum number of jobs to retain in memory
func (m *JobManager) SetMaxJobs(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if max > 0 {
		m.maxJobs = max
	}
}
