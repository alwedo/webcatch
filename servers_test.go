package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewCapture(t *testing.T) {
	store := NewCallStore()
	server := NewCapture(store, ":8080")

	if server == nil {
		t.Fatal("NewCapture returned nil")
	}
	if server.Addr != ":8080" {
		t.Errorf("expected addr :8080, got %s", server.Addr)
	}
}

func TestCapture_HandlesRequest(t *testing.T) {
	store := NewCallStore()
	server := NewCapture(store, ":8080")

	req := httptest.NewRequest("POST", "/test-path", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	calls := store.GetAll()
	if len(calls) != 1 {
		t.Fatalf("expected 1 captured call, got %d", len(calls))
	}

	call := calls[0]
	if call.Method != "POST" {
		t.Errorf("expected method POST, got %s", call.Method)
	}
	if call.Path != "/test-path" {
		t.Errorf("expected path /test-path, got %s", call.Path)
	}
	if !strings.Contains(call.Body, `{"key":"value"}`) {
		t.Errorf("expected body to contain {\"key\":\"value\"}, got %s", call.Body)
	}
}

func TestNewViewer(t *testing.T) {
	store := NewCallStore()
	server := NewViewer(store, ":8081")

	if server == nil {
		t.Fatal("NewViewer returned nil")
	}
	if server.Addr != ":8081" {
		t.Errorf("expected addr :8081, got %s", server.Addr)
	}
}

func TestViewer_RendersHTML(t *testing.T) {
	store := NewCallStore()
	server := NewViewer(store, ":8081")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected content-type text/html, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "webcatch") {
		t.Error("expected body to contain 'webcatch'")
	}
}

func TestViewer_Clear(t *testing.T) {
	store := NewCallStore()
	store.Add(CapturedCall{Method: "GET", Path: "/test"})
	server := NewViewer(store, ":8081")

	if len(store.GetAll()) != 1 {
		t.Fatalf("expected 1 call before clear, got %d", len(store.GetAll()))
	}

	req := httptest.NewRequest("POST", "/clear", nil)
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", w.Code)
	}

	if len(store.GetAll()) != 0 {
		t.Errorf("expected 0 calls after clear, got %d", len(store.GetAll()))
	}
}

func TestViewer_SSEEvents(t *testing.T) {
	store := NewCallStore()
	server := NewViewer(store, ":8081")

	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()

	done := make(chan bool)
	go func() {
		server.Handler.ServeHTTP(w, req)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	store.Add(CapturedCall{Method: "GET", Path: "/test"})

	time.Sleep(50 * time.Millisecond)

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected content-type text/event-stream, got %s", contentType)
	}
}
