package service

import (
	"context"
	"sync"
)

type ActiveRuns struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewActiveRuns() *ActiveRuns {
	return &ActiveRuns{
		cancels: make(map[string]context.CancelFunc),
	}
}

func (a *ActiveRuns) Set(runID string, cancel context.CancelFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cancels[runID] = cancel
}

func (a *ActiveRuns) Cancel(runID string) bool {
	a.mu.Lock()
	cancel, ok := a.cancels[runID]
	a.mu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

func (a *ActiveRuns) Has(runID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.cancels[runID]
	return ok
}

func (a *ActiveRuns) Delete(runID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.cancels, runID)
}
