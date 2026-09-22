package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/alwedo/webcatch/assets"
)

func NewCapture(store *CallStore, addr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 0)
		if r.Body != nil {
			buf := make([]byte, 1024*1024)
			n, err := r.Body.Read(buf)
			if err == nil || n > 0 {
				body = buf[:n]
			}
			r.Body.Close()
		}

		call := CapturedCall{
			Timestamp:  time.Now(),
			Method:     r.Method,
			Path:       r.URL.String(),
			Headers:    r.Header,
			Body:       string(body),
			RemoteAddr: r.RemoteAddr,
		}

		store.Add(call)

		w.WriteHeader(http.StatusOK)
	})

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func NewViewer(store *CallStore, addr string) *http.Server {
	tmpl := template.Must(template.ParseFS(assets.HTMLFiles, "template.tmpl"))

	mux := http.NewServeMux()

	fileServer := http.FileServerFS(assets.StaticFiles)
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/", handleIndex(store, tmpl))
	mux.HandleFunc("/events", handleEvents(store))
	mux.HandleFunc("POST /calls/{id}/viewed", handleViewed(store, tmpl))
	mux.HandleFunc("POST /calls/{id}/beautify", handleBeautify(store, tmpl, true))
	mux.HandleFunc("POST /calls/{id}/minify", handleBeautify(store, tmpl, false))
	mux.HandleFunc("POST /clear", handleClear(store, tmpl))

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func renderTemplate(tmpl *template.Template, w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
	}
}

// pageData is the data for templates that need the call list and the current
// sort order.
type pageData struct {
	Calls []CapturedCall
	Sort  string // "desc" (newest first, default) or "asc" (oldest first)
}

// sortParam returns the requested sort order from the query string, falling
// back to the browser URL reported by htmx (HX-Current-URL) so that
// SSE-triggered re-renders preserve the current sort.
func sortParam(r *http.Request) string {
	if s := r.URL.Query().Get("sort"); s != "" {
		return s
	}
	if cur := r.Header.Get("HX-Current-URL"); cur != "" {
		if u, err := url.Parse(cur); err == nil {
			return u.Query().Get("sort")
		}
	}
	return ""
}

func handleIndex(store *CallStore, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		calls := store.GetAll()
		data := pageData{Calls: calls, Sort: "desc"}
		if sortParam(r) == "asc" {
			slices.Reverse(calls)
			data.Sort = "asc"
		}
		renderTemplate(tmpl, w, "view", data)
	}
}

func handleViewed(store *CallStore, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid call id", http.StatusBadRequest)
			return
		}

		call, ok := store.MarkViewed(id)
		if !ok {
			http.Error(w, "call not found", http.StatusNotFound)
			return
		}

		renderTemplate(tmpl, w, "call-card", call)
	}
}

func handleBeautify(store *CallStore, tmpl *template.Template, beautified bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid call id", http.StatusBadRequest)
			return
		}

		call, ok := store.SetBeautified(id, beautified)
		if !ok {
			http.Error(w, "call not found", http.StatusNotFound)
			return
		}

		renderTemplate(tmpl, w, "call-card", call)
	}
}

func handleEvents(store *CallStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ch := store.Subscribe()
		defer store.Unsubscribe(ch)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case _, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "event: new-call\ndata: new-call\n\n")
				flusher.Flush()
			}
		}
	}
}

func handleClear(store *CallStore, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.Clear()

		if r.Header.Get("HX-Request") == "true" {
			data := pageData{Calls: store.GetAll(), Sort: "desc"}
			if sortParam(r) == "asc" {
				data.Sort = "asc"
			}
			renderTemplate(tmpl, w, "page-content", data)
			return
		}

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	}
}
