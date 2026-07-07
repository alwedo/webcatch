package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var Version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("webcatch %s\n", Version)
		os.Exit(0)
	}

	store := NewCallStore()

	captureServer := NewCapture(store, ":8080")
	viewerServer := NewViewer(store, ":8081")

	var wg sync.WaitGroup

	wg.Go(func() {
		log.Println("Capture server listening on :8080")
		if err := captureServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Capture server failed: %v", err)
		}
	})

	wg.Go(func() {
		log.Println("View server listening on :8081")
		if err := viewerServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("View server failed: %v", err)
		}
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down servers...")

	store.CloseListeners()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := captureServer.Shutdown(ctx); err != nil {
		log.Printf("Capture server shutdown error: %v", err)
	}

	if err := viewerServer.Shutdown(ctx); err != nil {
		log.Printf("Viewer server shutdown error: %v", err)
	}

	wg.Wait()
	log.Println("Servers stopped gracefully")
}
