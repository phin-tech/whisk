package server

import "context"

const ptyOutputBatchMaxBytes = 64 * 1024

type ptyOutputSegment struct {
	offset uint64
	bytes  []byte
}

type ptyOutputBatchOptions struct {
	FlushImmediately bool
	FlushMaxBytes    int
}

type ptyOutputBatchWriter func(context.Context, ptyOutputSegment) error

type ptyOutputBatcher struct {
	write  ptyOutputBatchWriter
	offset uint64
	bytes  []byte
}

func newPTYOutputBatcher(write ptyOutputBatchWriter) *ptyOutputBatcher {
	return &ptyOutputBatcher{
		write: write,
	}
}

func (b *ptyOutputBatcher) Enqueue(ctx context.Context, segment ptyOutputSegment, opts ptyOutputBatchOptions) error {
	if len(segment.bytes) == 0 {
		return nil
	}
	if len(b.bytes) > 0 && segment.offset != b.offset+uint64(len(b.bytes)) {
		if err := b.Flush(ctx); err != nil {
			return err
		}
	}
	if len(b.bytes) == 0 {
		b.offset = segment.offset
	}
	b.bytes = append(b.bytes, segment.bytes...)
	if opts.FlushImmediately && b.withinFlushLimit(opts.FlushMaxBytes) {
		return b.Flush(ctx)
	}
	if len(b.bytes) >= ptyOutputBatchMaxBytes {
		return b.Flush(ctx)
	}
	return nil
}

func (b *ptyOutputBatcher) Flush(ctx context.Context) error {
	if len(b.bytes) == 0 {
		return nil
	}
	segment := ptyOutputSegment{
		offset: b.offset,
		bytes:  b.bytes,
	}
	b.offset = 0
	b.bytes = nil
	return b.write(ctx, segment)
}

func (b *ptyOutputBatcher) withinFlushLimit(maxBytes int) bool {
	return maxBytes <= 0 || len(b.bytes) <= maxBytes
}
