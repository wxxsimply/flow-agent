package web

import "sync"

var runWorkflowMu sync.Map // runID -> *sync.Mutex
var runWorkflowActive sync.Map // runID -> struct{}

func lockRunWorkflow(runID string) func() {
	runWorkflowActive.Store(runID, struct{}{})
	v, _ := runWorkflowMu.LoadOrStore(runID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return func() {
		mu.Unlock()
		runWorkflowActive.Delete(runID)
	}
}

// IsRunWorkflowActive reports whether a background workflow goroutine holds the lock for runID.
func IsRunWorkflowActive(runID string) bool {
	_, ok := runWorkflowActive.Load(runID)
	return ok
}

func tryLockRunWorkflow(runID string) (unlock func(), ok bool) {
	v, _ := runWorkflowMu.LoadOrStore(runID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	if !mu.TryLock() {
		return nil, false
	}
	return mu.Unlock, true
}
