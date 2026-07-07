package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewCallStore(t *testing.T) {
	store := NewCallStore()
	if store == nil {
		t.Fatal("NewCallStore returned nil")
	}
	if store.calls == nil {
		t.Error("calls slice not initialized")
	}
	if len(store.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(store.calls))
	}
}

func TestCallStore_Add(t *testing.T) {
	store := NewCallStore()

	call := CapturedCall{
		Timestamp:  time.Now(),
		Method:     "POST",
		Path:       "/test",
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       `{"key": "value"}`,
		RemoteAddr: "127.0.0.1:1234",
	}

	store.Add(call)

	calls := store.GetAll()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	if calls[0].Method != "POST" {
		t.Errorf("expected method POST, got %s", calls[0].Method)
	}
	if calls[0].Path != "/test" {
		t.Errorf("expected path /test, got %s", calls[0].Path)
	}
	if calls[0].Body != `{"key": "value"}` {
		t.Errorf("expected body {\"key\": \"value\"}, got %s", calls[0].Body)
	}
}

func TestCallStore_GetAll_Reversed(t *testing.T) {
	store := NewCallStore()

	call1 := CapturedCall{Timestamp: time.Now(), Method: "GET", Path: "/first"}
	call2 := CapturedCall{Timestamp: time.Now().Add(time.Second), Method: "POST", Path: "/second"}
	call3 := CapturedCall{Timestamp: time.Now().Add(2 * time.Second), Method: "PUT", Path: "/third"}

	store.Add(call1)
	store.Add(call2)
	store.Add(call3)

	calls := store.GetAll()
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(calls))
	}

	if calls[0].Path != "/third" {
		t.Errorf("expected first call to be /third (reversed), got %s", calls[0].Path)
	}
	if calls[1].Path != "/second" {
		t.Errorf("expected second call to be /second, got %s", calls[1].Path)
	}
	if calls[2].Path != "/first" {
		t.Errorf("expected third call to be /first, got %s", calls[2].Path)
	}
}

func TestCallStore_Clear(t *testing.T) {
	store := NewCallStore()

	store.Add(CapturedCall{Method: "GET", Path: "/test1"})
	store.Add(CapturedCall{Method: "POST", Path: "/test2"})

	if len(store.GetAll()) != 2 {
		t.Fatalf("expected 2 calls before clear, got %d", len(store.GetAll()))
	}

	store.Clear()

	calls := store.GetAll()
	if len(calls) != 0 {
		t.Errorf("expected 0 calls after clear, got %d", len(calls))
	}
}

func TestCallStore_Subscribe_Unsubscribe(t *testing.T) {
	store := NewCallStore()

	ch := store.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	done := make(chan bool)
	go func() {
		select {
		case call := <-ch:
			if call.Method != "DELETE" {
				t.Errorf("expected method DELETE, got %s", call.Method)
			}
			done <- true
		case <-time.After(100 * time.Millisecond):
			t.Error("timeout waiting for call notification")
			done <- false
		}
	}()

	store.Add(CapturedCall{Method: "DELETE", Path: "/test"})

	<-done
	store.Unsubscribe(ch)
}

func TestCallStore_MultipleSubscribers(t *testing.T) {
	store := NewCallStore()

	ch1 := store.Subscribe()
	ch2 := store.Subscribe()

	done := make(chan int, 2)

	listen := func(ch chan CapturedCall, id int) {
		select {
		case call := <-ch:
			if call.Method != "PATCH" {
				t.Errorf("subscriber %d: expected method PATCH, got %s", id, call.Method)
			}
			done <- id
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for call notification", id)
			done <- -1
		}
	}

	go listen(ch1, 1)
	go listen(ch2, 2)

	store.Add(CapturedCall{Method: "PATCH", Path: "/test"})

	<-done
	<-done

	store.Unsubscribe(ch1)
	store.Unsubscribe(ch2)
}

func TestCallStore_ConcurrentAdds(t *testing.T) {
	store := NewCallStore()

	done := make(chan bool)

	add := func(path string) {
		for range 100 {
			store.Add(CapturedCall{Method: "GET", Path: path})
		}
		done <- true
	}

	go add("/path1")
	go add("/path2")
	go add("/path3")

	<-done
	<-done
	<-done

	calls := store.GetAll()
	if len(calls) != 300 {
		t.Errorf("expected 300 calls, got %d", len(calls))
	}
}
