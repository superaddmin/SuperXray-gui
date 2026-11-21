package websocket

import (
	"sync"
	"testing"
	"time"
)

func TestBroadcastRemovesSlowClientsAndKeepsDelivering(t *testing.T) {
	h := NewHub()
	live := &Client{Send: make(chan []byte, 1)}
	h.Register(live)
	for range 110 {
		h.Register(&Client{Send: make(chan []byte)})
	}
	done := make(chan struct{})
	go func() {
		h.Run()
		close(done)
	}()
	t.Cleanup(func() {
		h.Stop()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("hub did not stop")
		}
	})
	for range 2 {
		h.Broadcast(MessageTypeStatus, "online")
		select {
		case <-live.Send:
		case <-time.After(3 * time.Second):
			t.Fatal("live client missed broadcast")
		}
	}
	if got := h.GetClientCount(); got != 1 {
		t.Fatalf("remaining clients = %d, want 1", got)
	}
}

func TestTopicDeliveryAndUnregister(t *testing.T) {
	h := NewHub()
	defer h.Stop()
	receiver := &Client{Send: make(chan []byte, 2), Topics: map[MessageType]bool{MessageTypeTraffic: true}}
	h.Register(receiver)
	h.BroadcastToTopic(MessageTypeStatus, "online")
	h.BroadcastToTopic(MessageTypeTraffic, "updated")
	if len(receiver.Send) != 1 {
		t.Fatal("topic selection queued an unexpected number of messages")
	}
	<-receiver.Send
	var wg sync.WaitGroup
	wg.Go(func() { h.BroadcastToTopic(MessageTypeTraffic, "updated") })
	wg.Go(func() { h.Unregister(receiver) })
	wg.Wait()
	h.Register(receiver)
	if h.GetClientCount() != 0 {
		t.Fatal("closed client was registered again")
	}
	h.Stop()
	late := &Client{Send: make(chan []byte)}
	h.Register(late)
	if _, open := <-late.Send; open {
		t.Fatal("client accepted after shutdown")
	}
}
