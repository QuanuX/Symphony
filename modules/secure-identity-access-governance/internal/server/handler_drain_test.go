package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerDrainRetainsCustodyUntilActualWorkStops(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	handler := &drainingHandler{next: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { close(entered); <-release })}
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
		close(finished)
	}()
	<-entered
	drained := make(chan struct{})
	go func() { handler.drain(); close(drained) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		handler.mu.Lock()
		closed := handler.closing
		handler.mu.Unlock()
		if closed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("drain admission barrier timed out")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-drained:
		t.Fatal("drain released custody with handler still running")
	default:
	}
	late := httptest.NewRecorder()
	handler.ServeHTTP(late, httptest.NewRequest("GET", "/", nil))
	if late.Code != http.StatusServiceUnavailable {
		t.Fatalf("late request entered stopped owner: %d", late.Code)
	}
	close(release)
	select {
	case <-drained:
	case <-time.After(5 * time.Second):
		t.Fatal("completed work did not release custody")
	}
	<-finished
}
