package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

//go:embed template.html
var templateHTML string

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
	mux := http.NewServeMux()
	tmpl := template.Must(template.New("view").Parse(templateHTML))

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := tmpl.Execute(w, store.GetAll()); err != nil {
			http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
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
				fmt.Fprintf(w, "data: new-call\n\n")
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("POST /clear", func(w http.ResponseWriter, _ *http.Request) {
		store.Clear()
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	})

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}
