package server

import (
	"context"
	"testing"
)

func TestPTYOutputBatcherCoalescesContiguousOutput(t *testing.T) {
	ctx := context.Background()
	var writes []ptyOutputSegment
	batcher := newPTYOutputBatcher(func(_ context.Context, segment ptyOutputSegment) error {
		writes = append(writes, ptyOutputSegment{
			offset: segment.offset,
			bytes:  append([]byte(nil), segment.bytes...),
		})
		return nil
	})

	if err := batcher.Enqueue(ctx, ptyOutputSegment{offset: 0, bytes: []byte("a")}, ptyOutputBatchOptions{}); err != nil {
		t.Fatalf("enqueue first segment: %v", err)
	}
	if err := batcher.Enqueue(ctx, ptyOutputSegment{offset: 1, bytes: []byte("b")}, ptyOutputBatchOptions{}); err != nil {
		t.Fatalf("enqueue second segment: %v", err)
	}
	if len(writes) != 0 {
		t.Fatalf("writes before flush = %#v", writes)
	}
	if err := batcher.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(writes) != 1 || writes[0].offset != 0 || string(writes[0].bytes) != "ab" {
		t.Fatalf("writes = %#v", writes)
	}
}

func TestPTYOutputBatcherFlushesSmallInteractiveOutput(t *testing.T) {
	ctx := context.Background()
	var writes []ptyOutputSegment
	batcher := newPTYOutputBatcher(func(_ context.Context, segment ptyOutputSegment) error {
		writes = append(writes, ptyOutputSegment{
			offset: segment.offset,
			bytes:  append([]byte(nil), segment.bytes...),
		})
		return nil
	})

	opts := ptyOutputBatchOptions{FlushImmediately: true, FlushMaxBytes: 1024}
	if err := batcher.Enqueue(ctx, ptyOutputSegment{offset: 0, bytes: []byte("\x1b[20;2Hredraw")}, opts); err != nil {
		t.Fatalf("enqueue interactive segment: %v", err)
	}
	if len(writes) != 1 || writes[0].offset != 0 || string(writes[0].bytes) != "\x1b[20;2Hredraw" {
		t.Fatalf("writes = %#v", writes)
	}
}

func TestPTYOutputBatcherKeepsLargeInteractiveBurstBatched(t *testing.T) {
	ctx := context.Background()
	var writes []ptyOutputSegment
	batcher := newPTYOutputBatcher(func(_ context.Context, segment ptyOutputSegment) error {
		writes = append(writes, ptyOutputSegment{
			offset: segment.offset,
			bytes:  append([]byte(nil), segment.bytes...),
		})
		return nil
	})

	pending := make([]byte, 1020)
	for i := range pending {
		pending[i] = 'x'
	}
	if err := batcher.Enqueue(ctx, ptyOutputSegment{offset: 0, bytes: pending}, ptyOutputBatchOptions{}); err != nil {
		t.Fatalf("enqueue pending segment: %v", err)
	}
	opts := ptyOutputBatchOptions{FlushImmediately: true, FlushMaxBytes: 1024}
	if err := batcher.Enqueue(ctx, ptyOutputSegment{offset: 1020, bytes: []byte("redraw")}, opts); err != nil {
		t.Fatalf("enqueue interactive segment: %v", err)
	}
	if len(writes) != 0 {
		t.Fatalf("writes before interval flush = %#v", writes)
	}
	if err := batcher.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(writes) != 1 || writes[0].offset != 0 || len(writes[0].bytes) != 1026 {
		t.Fatalf("writes = %#v", writes)
	}
}

func TestPTYOutputBatcherFlushesBeforeOffsetGap(t *testing.T) {
	ctx := context.Background()
	var writes []ptyOutputSegment
	batcher := newPTYOutputBatcher(func(_ context.Context, segment ptyOutputSegment) error {
		writes = append(writes, ptyOutputSegment{
			offset: segment.offset,
			bytes:  append([]byte(nil), segment.bytes...),
		})
		return nil
	})

	if err := batcher.Enqueue(ctx, ptyOutputSegment{offset: 0, bytes: []byte("a")}, ptyOutputBatchOptions{}); err != nil {
		t.Fatalf("enqueue first segment: %v", err)
	}
	if err := batcher.Enqueue(ctx, ptyOutputSegment{offset: 2, bytes: []byte("b")}, ptyOutputBatchOptions{}); err != nil {
		t.Fatalf("enqueue gapped segment: %v", err)
	}
	if len(writes) != 1 || writes[0].offset != 0 || string(writes[0].bytes) != "a" {
		t.Fatalf("writes after gap = %#v", writes)
	}
	if err := batcher.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(writes) != 2 || writes[1].offset != 2 || string(writes[1].bytes) != "b" {
		t.Fatalf("writes = %#v", writes)
	}
}
