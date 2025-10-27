package timer

import (
	"container/heap"
	"sync"
	"time"
)

// TimerTask represents a task scheduled for future execution
type TimerTask struct {
	ID       string
	ExpiryAt time.Time
	Callback func()
	index    int // index in the heap (for heap.Interface)
}

// timerHeap is a min-heap of TimerTasks ordered by ExpiryAt
type timerHeap []*TimerTask

func (h timerHeap) Len() int { return len(h) }

func (h timerHeap) Less(i, j int) bool {
	return h[i].ExpiryAt.Before(h[j].ExpiryAt)
}

func (h timerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *timerHeap) Push(x interface{}) {
	n := len(*h)
	task := x.(*TimerTask)
	task.index = n
	*h = append(*h, task)
}

func (h *timerHeap) Pop() interface{} {
	old := *h
	n := len(old)
	task := old[n-1]
	old[n-1] = nil  // avoid memory leak
	task.index = -1 // for safety
	*h = old[0 : n-1]
	return task
}

// TimerManager manages scheduled tasks using a min-heap
type TimerManager struct {
	heap     timerHeap
	mu       sync.Mutex
	wakeup   chan struct{}
	tasks    map[string]*TimerTask // for O(1) lookup by ID
	workers  int
	workerWg sync.WaitGroup
	stopped  bool
	stopCh   chan struct{}
}

// NewTimerManager creates a new timer manager with a worker pool
func NewTimerManager(workers int) *TimerManager {
	tm := &TimerManager{
		heap:    make(timerHeap, 0),
		wakeup:  make(chan struct{}, 1),
		tasks:   make(map[string]*TimerTask),
		workers: workers,
		stopCh:  make(chan struct{}),
	}
	heap.Init(&tm.heap)
	return tm
}

// Start starts the timer manager and its worker pool
func (tm *TimerManager) Start() {
	// Start worker goroutines
	for i := 0; i < tm.workers; i++ {
		tm.workerWg.Add(1)
		go tm.worker()
	}

	// Start the main scheduler goroutine
	go tm.run()
}

// Stop stops the timer manager gracefully
func (tm *TimerManager) Stop() {
	tm.mu.Lock()
	if tm.stopped {
		tm.mu.Unlock()
		return
	}
	tm.stopped = true
	close(tm.stopCh)
	tm.mu.Unlock()

	// Wait for workers to finish
	tm.workerWg.Wait()
}

// Schedule adds a new task to be executed at the specified time
func (tm *TimerManager) Schedule(id string, expiryAt time.Time, callback func()) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.stopped {
		return ErrManagerStopped
	}

	// Remove existing task with same ID if present
	if existing, ok := tm.tasks[id]; ok {
		heap.Remove(&tm.heap, existing.index)
		delete(tm.tasks, id)
	}

	task := &TimerTask{
		ID:       id,
		ExpiryAt: expiryAt,
		Callback: callback,
	}

	heap.Push(&tm.heap, task)
	tm.tasks[id] = task

	// Wake up the scheduler if this is the earliest task
	if tm.heap[0] == task {
		select {
		case tm.wakeup <- struct{}{}:
		default:
		}
	}

	return nil
}

// Cancel removes a scheduled task
func (tm *TimerManager) Cancel(id string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[id]
	if !ok {
		return false
	}

	heap.Remove(&tm.heap, task.index)
	delete(tm.tasks, id)
	return true
}

// run is the main scheduler loop
func (tm *TimerManager) run() {
	for {
		tm.mu.Lock()

		if tm.stopped {
			tm.mu.Unlock()
			return
		}

		var waitDuration time.Duration
		if tm.heap.Len() == 0 {
			// No tasks, wait indefinitely
			waitDuration = 24 * time.Hour
		} else {
			// Calculate wait time until next task
			nextTask := tm.heap[0]
			waitDuration = time.Until(nextTask.ExpiryAt)

			if waitDuration <= 0 {
				// Task is ready to execute
				task := heap.Pop(&tm.heap).(*TimerTask)
				delete(tm.tasks, task.ID)

				// Submit to worker pool (non-blocking)
				go task.Callback()

				tm.mu.Unlock()
				continue
			}
		}

		tm.mu.Unlock()

		// Wait for either timeout or wakeup signal
		timer := time.NewTimer(waitDuration)
		select {
		case <-timer.C:
			// Time to check for expired tasks
		case <-tm.wakeup:
			// New task added or existing task updated
			timer.Stop()
		case <-tm.stopCh:
			timer.Stop()
			return
		}
	}
}

// worker processes tasks from the task channel
func (tm *TimerManager) worker() {
	defer tm.workerWg.Done()

	<-tm.stopCh
}

// Stats returns statistics about the timer manager
func (tm *TimerManager) Stats() TimerStats {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	return TimerStats{
		ScheduledTasks: len(tm.tasks),
		Workers:        tm.workers,
	}
}

// TimerStats contains statistics about the timer manager
type TimerStats struct {
	ScheduledTasks int
	Workers        int
}

var (
	ErrManagerStopped = &TimerError{"timer manager is stopped"}
)

// TimerError represents a timer error
type TimerError struct {
	msg string
}

func (e *TimerError) Error() string {
	return e.msg
}
