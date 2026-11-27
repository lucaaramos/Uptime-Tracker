package ws

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestBroadcastRawJSON(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:  hub,
		send: make(chan []byte, 4),
	}

	hub.mu.Lock()
	hub.clients[client] = true
	hub.mu.Unlock()

	raw := []byte(`{"monitor_id":"abc123","success":true,"status_code":200,"latency_ms":42}`)
	hub.Broadcast(json.RawMessage(raw))

	got := <-client.send

	var decoded map[string]interface{}
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("expected valid JSON object, got %q: %v", string(got), err)
	}
	if decoded["monitor_id"] != "abc123" {
		t.Errorf("expected monitor_id abc123, got %v", decoded["monitor_id"])
	}
	if decoded["success"] != true {
		t.Errorf("expected success true, got %v", decoded["success"])
	}
}

func TestHubConcurrentBroadcastAndUnregister(t *testing.T) {
	hub := NewHub()

	const numClients = 20
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		// Small buffer (1) to trigger slow-client eviction under Broadcast
		c := &Client{
			hub:  hub,
			send: make(chan []byte, 1),
		}
		clients[i] = c
		hub.mu.Lock()
		hub.clients[c] = true
		hub.mu.Unlock()
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				hub.Broadcast(json.RawMessage(`{"ok":true}`))
			}
		}()
	}

	for _, c := range clients {
		wg.Add(1)
		go func(cl *Client) {
			defer wg.Done()
			hub.Unregister(cl)
			// Calling Unregister twice must not panic
			hub.Unregister(cl)
		}(c)
	}

	wg.Wait()

	hub.mu.Lock()
	remaining := len(hub.clients)
	hub.mu.Unlock()
	if remaining != 0 {
		t.Errorf("expected 0 remaining clients, got %d", remaining)
	}
}
