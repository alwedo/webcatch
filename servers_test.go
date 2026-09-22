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

func TestViewer_MarkViewed(t *testing.T) {
	store := NewCallStore()
	store.Add(CapturedCall{Method: "GET", Path: "/test"})
	server := NewViewer(store, ":8081")

	req := httptest.NewRequest("POST", "/calls/1/viewed", nil)
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "[ new ]") {
		t.Error("expected viewed card to have no [ new ] badge")
	}
	if !strings.Contains(body, "/test") {
		t.Error("expected card body to contain the call path")
	}

	calls := store.GetAll()
	if !calls[0].Viewed {
		t.Error("expected call to be marked viewed in the store")
	}
}

func TestViewer_MarkViewed_UnknownID(t *testing.T) {
	store := NewCallStore()
	store.Add(CapturedCall{Method: "GET", Path: "/test"})
	server := NewViewer(store, ":8081")

	req := httptest.NewRequest("POST", "/calls/999/viewed", nil)
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestViewer_MarkViewed_InvalidID(t *testing.T) {
	store := NewCallStore()
	server := NewViewer(store, ":8081")

	req := httptest.NewRequest("POST", "/calls/abc/viewed", nil)
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestViewer_Beautify(t *testing.T) {
	store := NewCallStore()
	store.Add(CapturedCall{Method: "GET", Path: "/test", Body: `{"key":"value"}`})
	server := NewViewer(store, ":8081")

	// The page shows the beautify button for JSON bodies.
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(w.Body.String(), "beautify json") {
		t.Error("expected page to show beautify button for JSON body")
	}

	// Beautifying returns the prettified card and flips the button label.
	// (html/template escapes quotes as &#34; in text contexts.)
	w = httptest.NewRecorder()
	server.Handler.ServeHTTP(w, httptest.NewRequest("POST", "/calls/1/beautify", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `&#34;key&#34;: &#34;value&#34;`) {
		t.Errorf("expected prettified body (spaced colon) in response, got %s", body)
	}
	if strings.Contains(body, `{&#34;key&#34;:&#34;value&#34;}`) {
		t.Errorf("expected body not to be compact after beautify, got %s", body)
	}
	if !strings.Contains(body, "minify json") {
		t.Error("expected button label to flip to minify after beautify")
	}

	// Minifying returns the compact body again.
	w = httptest.NewRecorder()
	server.Handler.ServeHTTP(w, httptest.NewRequest("POST", "/calls/1/minify", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `{&#34;key&#34;:&#34;value&#34;}`) {
		t.Errorf("expected compact body after minify, got %s", w.Body.String())
	}
}

func TestViewer_Beautify_NonJSONBody(t *testing.T) {
	store := NewCallStore()
	store.Add(CapturedCall{Method: "GET", Path: "/test", Body: "plain text"})
	server := NewViewer(store, ":8081")

	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if strings.Contains(w.Body.String(), "json-beautify-btn") {
		t.Error("expected no beautify button for non-JSON body")
	}
}

func TestViewer_Beautify_UnknownID(t *testing.T) {
	store := NewCallStore()
	server := NewViewer(store, ":8081")

	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, httptest.NewRequest("POST", "/calls/999/beautify", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestViewer_SortOrder(t *testing.T) {
	store := NewCallStore()
	store.Add(CapturedCall{Method: "GET", Path: "/first"})
	store.Add(CapturedCall{Method: "GET", Path: "/second"})
	server := NewViewer(store, ":8081")

	tests := []struct {
		name       string
		query      string
		currentURL string
		wantFirst  string
		wantToggle string // label reflects the current sort state
	}{
		{name: "default is newest first", query: "", wantFirst: "/second", wantToggle: "newest first"},
		{name: "explicit desc", query: "?sort=desc", wantFirst: "/second", wantToggle: "newest first"},
		{name: "asc via query", query: "?sort=asc", wantFirst: "/first", wantToggle: "oldest first"},
		{name: "asc via HX-Current-URL", currentURL: "http://localhost:8081/?sort=asc", wantFirst: "/first", wantToggle: "oldest first"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tt.query, nil)
			if tt.currentURL != "" {
				req.Header.Set("HX-Current-URL", tt.currentURL)
			}
			w := httptest.NewRecorder()
			server.Handler.ServeHTTP(w, req)

			body := w.Body.String()
			firstIdx := strings.Index(body, tt.wantFirst)
			if firstIdx == -1 {
				t.Fatalf("expected body to contain %s, got %s", tt.wantFirst, body)
			}
			other := "/first"
			if tt.wantFirst == "/first" {
				other = "/second"
			}
			if idx := strings.Index(body, other); idx != -1 && idx < firstIdx {
				t.Errorf("expected %s before %s, got %s", tt.wantFirst, other, body)
			}
			if !strings.Contains(body, tt.wantToggle) {
				t.Errorf("expected sort toggle to show %q, got %s", tt.wantToggle, body)
			}
		})
	}
}

func TestViewer_Clear_HTMXRequest(t *testing.T) {
	store := NewCallStore()
	store.Add(CapturedCall{Method: "GET", Path: "/test"})
	server := NewViewer(store, ":8081")

	req := httptest.NewRequest("POST", "/clear", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if len(store.GetAll()) != 0 {
		t.Errorf("expected 0 calls after clear, got %d", len(store.GetAll()))
	}

	body := w.Body.String()
	if !strings.Contains(body, `id="content"`) {
		t.Error("expected fragment to contain the content wrapper div")
	}
	if !strings.Contains(body, "no calls captured") {
		t.Error("expected fragment to contain the empty state")
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
