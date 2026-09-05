package editlease

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHubDeliversPublishedTokenToSubscriber(t *testing.T) {
	hub := NewHub()
	documentID := uuid.New()

	id, events := hub.Subscribe(documentID)
	defer hub.Unsubscribe(documentID, id)

	hub.Publish(documentID, "token-a")

	select {
	case token := <-events:
		if token != "token-a" {
			t.Fatalf("received token = %q, want %q", token, "token-a")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for published token")
	}
}

func TestHubDoesNotDeliverAcrossDocuments(t *testing.T) {
	hub := NewHub()
	documentA := uuid.New()
	documentB := uuid.New()

	_, eventsA := hub.Subscribe(documentA)
	hub.Publish(documentB, "token-b")

	select {
	case token := <-eventsA:
		t.Fatalf("subscriber for document A unexpectedly received %q from a publish to document B", token)
	case <-time.After(50 * time.Millisecond):
		// Expected: nothing arrives.
	}
}

func TestHubFansOutToEverySubscriberOfTheSameDocument(t *testing.T) {
	hub := NewHub()
	documentID := uuid.New()

	id1, events1 := hub.Subscribe(documentID)
	id2, events2 := hub.Subscribe(documentID)
	defer hub.Unsubscribe(documentID, id1)
	defer hub.Unsubscribe(documentID, id2)

	hub.Publish(documentID, "token-shared")

	for _, events := range []<-chan string{events1, events2} {
		select {
		case token := <-events:
			if token != "token-shared" {
				t.Fatalf("received token = %q, want %q", token, "token-shared")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for fan-out delivery")
		}
	}
}

func TestHubUnsubscribeClosesChannelAndStopsDelivery(t *testing.T) {
	hub := NewHub()
	documentID := uuid.New()

	id, events := hub.Subscribe(documentID)
	hub.Unsubscribe(documentID, id)

	if _, ok := <-events; ok {
		t.Fatal("expected channel to be closed after Unsubscribe")
	}

	// Publishing after every subscriber left should be a no-op, not a panic.
	hub.Publish(documentID, "token-after-unsubscribe")
}

func TestHubPublishDoesNotBlockWhenSubscriberBufferIsFull(t *testing.T) {
	hub := NewHub()
	documentID := uuid.New()

	id, events := hub.Subscribe(documentID)
	defer hub.Unsubscribe(documentID, id)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// The subscriber channel has a small fixed buffer; publishing well
		// past it must never block the publisher (a claim request in
		// progress), even though nothing is draining `events` yet.
		for i := 0; i < 50; i++ {
			hub.Publish(documentID, "token")
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked instead of dropping messages for a full subscriber")
	}

	// Drain whatever made it into the buffer so the deferred Unsubscribe's
	// channel close doesn't race with a blocked send (there shouldn't be one).
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}
