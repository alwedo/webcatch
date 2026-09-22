package main

import (
	"fmt"
	"html/template"
	"net/http"
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
	mux := http.NewServeMux()
	tmpl := template.Must(template.ParseFS(assets.HTMLFiles, "template.tmpl"))

	renderTemplate := func(w http.ResponseWriter, name string, data any) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
			http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
		}
	}

	fileServer := http.FileServerFS(assets.StaticFiles)
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		renderTemplate(w, "view", store.GetAll())
	})

	mux.HandleFunc("POST /calls/{id}/viewed", func(w http.ResponseWriter, r *http.Request) {
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

		renderTemplate(w, "call-card", call)
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
				fmt.Fprintf(w, "event: new-call\ndata: new-call\n\n")
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("POST /clear", func(w http.ResponseWriter, r *http.Request) {
		store.Clear()

		if r.Header.Get("HX-Request") == "true" {
			renderTemplate(w, "page-content", store.GetAll())
			return
		}

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	})

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}
