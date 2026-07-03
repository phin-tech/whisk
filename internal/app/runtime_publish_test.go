package app

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRuntimePublishRetainsEventsInSequenceOrder(t *testing.T) {
	sink := newBlockingRetainedEventSink()
	runtime := NewRuntime(RuntimeConfig{EventSink: sink})
	ctx := context.Background()

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		runtime.publish(ctx, RuntimeEvent{Type: EventSessionChanged})
	}()

	select {
	case <-sink.firstPublishStarted:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for first publish")
	}

	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		runtime.publish(ctx, RuntimeEvent{Type: EventPTYChanged})
	}()

	select {
	case <-sink.concurrentPublishStarted:
	case <-time.After(50 * time.Millisecond):
	}
	close(sink.releaseFirstPublish)

	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for first publish to finish")
	}
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for second publish to finish")
	}

	retained := sink.retainedEvents()
	if len(retained) != 2 {
		t.Fatalf("retained events = %#v", retained)
	}
	if retained[0].Seq != 1 || retained[1].Seq != 2 {
		t.Fatalf("retained event sequence order = %#v, want seq 1 then 2", retained)
	}
}

type blockingRetainedEventSink struct {
	mu                       sync.Mutex
	retained                 []RuntimeEvent
	firstPublishStarted      chan struct{}
	releaseFirstPublish      chan struct{}
	concurrentPublishStarted chan struct{}
	first                    bool
	firstStartedOnce         sync.Once
	concurrentStartedOnce    sync.Once
}

func newBlockingRetainedEventSink() *blockingRetainedEventSink {
	return &blockingRetainedEventSink{
		firstPublishStarted:      make(chan struct{}),
		releaseFirstPublish:      make(chan struct{}),
		concurrentPublishStarted: make(chan struct{}),
	}
}

func (s *blockingRetainedEventSink) Publish(_ context.Context, event RuntimeEvent) error {
	s.mu.Lock()
	if !s.first {
		s.first = true
		s.firstStartedOnce.Do(func() { close(s.firstPublishStarted) })
		s.mu.Unlock()
		<-s.releaseFirstPublish
	} else {
		s.concurrentStartedOnce.Do(func() { close(s.concurrentPublishStarted) })
		s.mu.Unlock()
	}

	s.mu.Lock()
	s.retained = append(s.retained, event)
	s.mu.Unlock()
	return nil
}

func (s *blockingRetainedEventSink) retainedEvents() []RuntimeEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]RuntimeEvent(nil), s.retained...)
}
