package main

import (
	"net/http"
	"slices"
	"sync"
	"time"
)

type CapturedCall struct {
	Timestamp  time.Time
	Method     string
	Path       string
	Headers    http.Header
	Body       string
	RemoteAddr string
}

type CallStore struct {
	mu        sync.RWMutex
	calls     []CapturedCall
	listeners []chan CapturedCall
}

func NewCallStore() *CallStore {
	return &CallStore{
		calls: make([]CapturedCall, 0),
	}
}

func (cs *CallStore) Add(call CapturedCall) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.calls = append(cs.calls, call)

	for _, listener := range cs.listeners {
		select {
		case listener <- call:
		default:
		}
	}
}

func (cs *CallStore) GetAll() []CapturedCall {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	result := make([]CapturedCall, len(cs.calls))
	copy(result, cs.calls)
	slices.Reverse(result)

	return result
}

func (cs *CallStore) Subscribe() chan CapturedCall {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	ch := make(chan CapturedCall, 10)
	cs.listeners = append(cs.listeners, ch)
	return ch
}

func (cs *CallStore) Unsubscribe(ch chan CapturedCall) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for i, listener := range cs.listeners {
		if listener == ch {
			cs.listeners = append(cs.listeners[:i], cs.listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

func (cs *CallStore) Clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.calls = make([]CapturedCall, 0)
}

func (cs *CallStore) CloseListeners() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for _, listener := range cs.listeners {
		close(listener)
	}
	cs.listeners = nil
}
