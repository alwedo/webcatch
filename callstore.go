package main

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

type CapturedCall struct {
	ID         int
	Timestamp  time.Time
	Method     string
	Path       string
	Headers    http.Header
	Body       string
	RemoteAddr string
	Viewed     bool
	Beautified bool
}

// DisplayBody returns the body prettified when the call is marked beautified,
// or the raw body otherwise. Reformatting copies JSON tokens stream-wise, so
// key order and number literals are preserved exactly.
func (c CapturedCall) DisplayBody() string {
	if !c.Beautified {
		return c.Body
	}

	var buf bytes.Buffer
	dec := jsontext.NewDecoder(strings.NewReader(c.Body))
	enc := jsontext.NewEncoder(&buf, jsontext.WithIndent("  "))

	for {
		switch dec.PeekKind() {
		case '{', '[', '}', ']':
			tok, err := dec.ReadToken()
			if err != nil {
				return c.Body
			}
			if err := enc.WriteToken(tok); err != nil {
				return c.Body
			}
		default: // scalar: copy the raw bytes to preserve number literals and escaping
			val, err := dec.ReadValue()
			if err != nil {
				if err == io.EOF {
					return strings.TrimSuffix(buf.String(), "\n") // enc terminates each value with a newline
				}
				return c.Body
			}
			if err := enc.WriteValue(val); err != nil {
				return c.Body
			}
		}
	}
}

// IsJSON reports whether the body is a single valid JSON value.
func (c CapturedCall) IsJSON() bool {
	dec := jsontext.NewDecoder(strings.NewReader(c.Body))
	if _, err := dec.ReadValue(); err != nil {
		return false
	}
	_, err := dec.ReadValue() // only trailing whitespace may remain
	return err == io.EOF
}

type CallStore struct {
	mu        sync.RWMutex
	calls     []CapturedCall
	listeners []chan CapturedCall
	nextID    int
}

func NewCallStore() *CallStore {
	return &CallStore{
		calls: make([]CapturedCall, 0),
	}
}

func (cs *CallStore) Add(call CapturedCall) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.nextID++
	call.ID = cs.nextID
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

func (cs *CallStore) MarkViewed(id int) (CapturedCall, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for i := range cs.calls {
		if cs.calls[i].ID == id {
			cs.calls[i].Viewed = true
			return cs.calls[i], true
		}
	}
	return CapturedCall{}, false
}

func (cs *CallStore) SetBeautified(id int, beautified bool) (CapturedCall, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for i := range cs.calls {
		if cs.calls[i].ID == id {
			cs.calls[i].Beautified = beautified
			return cs.calls[i], true
		}
	}
	return CapturedCall{}, false
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
