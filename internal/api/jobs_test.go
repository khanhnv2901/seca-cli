package api

import (
	"sync"
	"testing"
	"time"
)

func TestNewJobManager(t *testing.T) {
	jm := NewJobManager()
	if jm == nil {
		t.Fatal("expected non-nil JobManager")
	}
	if jm.maxJobs != 1000 {
		t.Errorf("expected maxJobs 1000, got %d", jm.maxJobs)
	}
	if jm.jobs == nil {
		t.Error("expected jobs map to be initialized")
	}
	if jm.subscribers == nil {
		t.Error("expected subscribers map to be initialized")
	}
}

func TestJobManager_CreateJob(t *testing.T) {
	jm := NewJobManager()

	job := jm.CreateJob("check", "result123")

	if job == nil {
		t.Fatal("expected non-nil job")
	}
	if job.Type != "check" {
		t.Errorf("expected type 'check', got %s", job.Type)
	}
	if job.ResultID != "result123" {
		t.Errorf("expected resultID 'result123', got %s", job.ResultID)
	}
	if job.Status != "pending" {
		t.Errorf("expected status 'pending', got %s", job.Status)
	}
	if job.ID == "" {
		t.Error("expected job to have an ID")
	}

	// Verify job was stored
	retrieved := jm.GetJob(job.ID)
	if retrieved == nil {
		t.Fatal("expected to retrieve created job")
	}
	if retrieved.ID != job.ID {
		t.Errorf("expected ID %s, got %s", job.ID, retrieved.ID)
	}
}

func TestJobManager_UpdateJob(t *testing.T) {
	jm := NewJobManager()
	job := jm.CreateJob("check", "result123")

	// Update job status
	updated := jm.UpdateJob(job.ID, func(j *Job) {
		j.Status = "running"
		now := time.Now()
		j.StartedAt = &now
	})

	if updated == nil {
		t.Fatal("expected non-nil updated job")
	}
	if updated.Status != "running" {
		t.Errorf("expected status 'running', got %s", updated.Status)
	}
	if updated.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	// Update non-existent job
	nonExistent := jm.UpdateJob("non-existent-id", func(j *Job) {
		j.Status = "completed"
	})
	if nonExistent != nil {
		t.Error("expected nil for non-existent job update")
	}
}

func TestJobManager_GetJob(t *testing.T) {
	jm := NewJobManager()

	// Get non-existent job
	job := jm.GetJob("non-existent")
	if job != nil {
		t.Error("expected nil for non-existent job")
	}

	// Create and get job
	created := jm.CreateJob("check", "result123")
	retrieved := jm.GetJob(created.ID)

	if retrieved == nil {
		t.Fatal("expected to retrieve job")
	}
	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}

	// Verify it returns a copy (not same pointer)
	if retrieved == created {
		t.Error("GetJob should return a copy, not the same pointer")
	}
}

func TestJobManager_ListJobs(t *testing.T) {
	jm := NewJobManager()

	// List empty jobs
	jobs := jm.ListJobs(10)
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	// Create multiple jobs with StartedAt times
	job1 := jm.CreateJob("check", "result1")
	start1 := time.Now()
	jm.UpdateJob(job1.ID, func(j *Job) {
		j.StartedAt = &start1
	})

	time.Sleep(10 * time.Millisecond)

	job2 := jm.CreateJob("verify", "result2")
	start2 := time.Now()
	jm.UpdateJob(job2.ID, func(j *Job) {
		j.StartedAt = &start2
	})

	time.Sleep(10 * time.Millisecond)

	job3 := jm.CreateJob("audit", "result3")
	start3 := time.Now()
	jm.UpdateJob(job3.ID, func(j *Job) {
		j.StartedAt = &start3
	})

	// List all jobs
	jobs = jm.ListJobs(10)
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	// Verify jobs are sorted by StartedAt (newest first)
	// job3 started last, so it should be first
	if jobs[0].ID != job3.ID {
		t.Errorf("expected first job to be job3 (%s), got %s", job3.ID, jobs[0].ID)
	}

	// Test limit
	jobs = jm.ListJobs(2)
	if len(jobs) != 2 {
		t.Errorf("expected limit to return 2 jobs, got %d", len(jobs))
	}
}

func TestJobManager_Subscribe(t *testing.T) {
	jm := NewJobManager()

	ch, unsubscribe := jm.Subscribe()
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	if unsubscribe == nil {
		t.Fatal("expected non-nil unsubscribe function")
	}

	// Create a job and verify subscriber receives it
	done := make(chan bool)
	go func() {
		select {
		case job := <-ch:
			if job.Type != "check" {
				t.Errorf("expected type 'check', got %s", job.Type)
			}
			done <- true
		case <-time.After(1 * time.Second):
			t.Error("timeout waiting for job notification")
			done <- false
		}
	}()

	jm.CreateJob("check", "result123")

	if !<-done {
		t.Error("failed to receive job notification")
	}

	// Unsubscribe and verify no more notifications
	unsubscribe()

	// Give it a moment to unsubscribe
	time.Sleep(50 * time.Millisecond)

	jm.CreateJob("check2", "result456")

	// Channel should be closed after unsubscribe
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after unsubscribe")
		}
	case <-time.After(100 * time.Millisecond):
		// No message received, which is expected
	}
}

func TestJobManager_Broadcast(t *testing.T) {
	jm := NewJobManager()

	// Subscribe multiple listeners
	ch1, unsub1 := jm.Subscribe()
	ch2, unsub2 := jm.Subscribe()
	defer unsub1()
	defer unsub2()

	var wg sync.WaitGroup
	wg.Add(2)

	received1 := false
	received2 := false

	go func() {
		defer wg.Done()
		select {
		case <-ch1:
			received1 = true
		case <-time.After(1 * time.Second):
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case <-ch2:
			received2 = true
		case <-time.After(1 * time.Second):
		}
	}()

	jm.CreateJob("check", "result123")
	wg.Wait()

	if !received1 {
		t.Error("subscriber 1 should have received notification")
	}
	if !received2 {
		t.Error("subscriber 2 should have received notification")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID("test")
	id2 := generateID("test")

	if id1 == "" {
		t.Error("expected non-empty ID")
	}
	if id2 == "" {
		t.Error("expected non-empty ID")
	}
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
	if id1[:4] != "test" {
		t.Errorf("expected ID to start with 'test', got %s", id1)
	}
}

func TestJobManager_SetMaxJobs(t *testing.T) {
	jm := NewJobManager()

	jm.SetMaxJobs(500)

	if jm.maxJobs != 500 {
		t.Errorf("expected maxJobs 500, got %d", jm.maxJobs)
	}
}

func TestJobManager_ConcurrentAccess(t *testing.T) {
	jm := NewJobManager()

	var wg sync.WaitGroup
	numRoutines := 10
	jobsPerRoutine := 10

	// Concurrent creates
	wg.Add(numRoutines)
	for i := 0; i < numRoutines; i++ {
		go func(n int) {
			defer wg.Done()
			for j := 0; j < jobsPerRoutine; j++ {
				jm.CreateJob("check", "result")
			}
		}(i)
	}
	wg.Wait()

	jobs := jm.ListJobs(1000)
	expected := numRoutines * jobsPerRoutine
	if len(jobs) != expected {
		t.Errorf("expected %d jobs, got %d", expected, len(jobs))
	}

	// Concurrent reads
	wg.Add(numRoutines)
	for i := 0; i < numRoutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				jm.ListJobs(10)
			}
		}()
	}
	wg.Wait()
}

func TestJobLifecycle(t *testing.T) {
	jm := NewJobManager()

	// Create job
	job := jm.CreateJob("check", "result123")
	if job.Status != "pending" {
		t.Errorf("new job should be pending, got %s", job.Status)
	}

	// Start job
	now := time.Now()
	updated := jm.UpdateJob(job.ID, func(j *Job) {
		j.Status = "running"
		j.StartedAt = &now
	})
	if updated.Status != "running" {
		t.Errorf("expected status running, got %s", updated.Status)
	}
	if updated.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	// Complete job
	finished := time.Now()
	completed := jm.UpdateJob(job.ID, func(j *Job) {
		j.Status = "completed"
		j.FinishedAt = &finished
	})
	if completed.Status != "completed" {
		t.Errorf("expected status completed, got %s", completed.Status)
	}
	if completed.FinishedAt == nil {
		t.Error("expected FinishedAt to be set")
	}

	// Verify final state
	final := jm.GetJob(job.ID)
	if final.Status != "completed" {
		t.Errorf("expected final status completed, got %s", final.Status)
	}
}
