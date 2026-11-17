package server

import (
	"net/http"
	"sync"
)

// drainingHandler prevents late requests from entering owned state after
// shutdown, and joins admitted handlers even when HTTP Shutdown times out.
// Its admission mutex makes WaitGroup.Add and the final Wait non-overlapping.
// A stuck handler retains the package lease until the native manager terminates
// the process; timeout alone is never evidence that package use has ended.
type drainingHandler struct {
	next    http.Handler
	mu      sync.Mutex
	closing bool
	active  sync.WaitGroup
}

func (h *drainingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		http.Error(w, "SSIAG is stopping", http.StatusServiceUnavailable)
		return
	}
	h.active.Add(1)
	h.mu.Unlock()
	defer h.active.Done()
	h.next.ServeHTTP(w, r)
}
func (h *drainingHandler) drain() {
	h.mu.Lock()
	h.closing = true
	h.mu.Unlock()
	h.active.Wait()
}
