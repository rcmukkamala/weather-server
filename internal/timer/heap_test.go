package timer

import (
	"sync"
	"testing"
	"time"
)

func TestTimerManager_Schedule(t *testing.T) {
	tm := NewTimerManager(2)
	tm.Start()
	defer tm.Stop()

	executed := false
	var mu sync.Mutex

	err := tm.Schedule("test1", time.Now().Add(100*time.Millisecond), func() {
		mu.Lock()
		executed = true
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if !executed {
		t.Error("Task was not executed")
	}
	mu.Unlock()
}

func TestTimerManager_Cancel(t *testing.T) {
	tm := NewTimerManager(2)
	tm.Start()
	defer tm.Stop()

	executed := false
	var mu sync.Mutex

	err := tm.Schedule("test1", time.Now().Add(100*time.Millisecond), func() {
		mu.Lock()
		executed = true
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	// Cancel the task
	cancelled := tm.Cancel("test1")
	if !cancelled {
		t.Error("Cancel returned false")
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if executed {
		t.Error("Task was executed despite being cancelled")
	}
	mu.Unlock()
}

func TestTimerManager_MultipleTasksOrdering(t *testing.T) {
	tm := NewTimerManager(2)
	tm.Start()
	defer tm.Stop()

	var results []int
	var mu sync.Mutex

	// Schedule tasks in reverse order
	tm.Schedule("task3", time.Now().Add(150*time.Millisecond), func() {
		mu.Lock()
		results = append(results, 3)
		mu.Unlock()
	})

	tm.Schedule("task1", time.Now().Add(50*time.Millisecond), func() {
		mu.Lock()
		results = append(results, 1)
		mu.Unlock()
	})

	tm.Schedule("task2", time.Now().Add(100*time.Millisecond), func() {
		mu.Lock()
		results = append(results, 2)
		mu.Unlock()
	})

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
	if results[0] != 1 || results[1] != 2 || results[2] != 3 {
		t.Errorf("Tasks executed in wrong order: %v", results)
	}
	mu.Unlock()
}

func TestTimerManager_RescheduleExisting(t *testing.T) {
	tm := NewTimerManager(2)
	tm.Start()
	defer tm.Stop()

	count := 0
	var mu sync.Mutex

	// Schedule a task
	tm.Schedule("test1", time.Now().Add(100*time.Millisecond), func() {
		mu.Lock()
		count++
		mu.Unlock()
	})

	// Reschedule with same ID (should replace)
	tm.Schedule("test1", time.Now().Add(50*time.Millisecond), func() {
		mu.Lock()
		count += 10
		mu.Unlock()
	})

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	if count != 10 {
		t.Errorf("Expected count=10 (only second task), got %d", count)
	}
	mu.Unlock()
}

func TestTimerManager_Stats(t *testing.T) {
	tm := NewTimerManager(5)
	tm.Start()
	defer tm.Stop()

	// Schedule some tasks
	tm.Schedule("task1", time.Now().Add(1*time.Hour), func() {})
	tm.Schedule("task2", time.Now().Add(2*time.Hour), func() {})
	tm.Schedule("task3", time.Now().Add(3*time.Hour), func() {})

	stats := tm.Stats()
	if stats.ScheduledTasks != 3 {
		t.Errorf("Expected 3 scheduled tasks, got %d", stats.ScheduledTasks)
	}
	if stats.Workers != 5 {
		t.Errorf("Expected 5 workers, got %d", stats.Workers)
	}
}
