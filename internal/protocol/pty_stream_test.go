package protocol

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestPTYBinaryOutputFrameRoundTrip(t *testing.T) {
	output := []byte("hello")
	offset := uint64(0x0102030405060708)

	frame := EncodePTYBinaryOutputFrame(offset, output)
	if got, want := frame[0], PTYBinaryOutputFrameKind; got != want {
		t.Fatalf("frame kind = 0x%02x, want 0x%02x", got, want)
	}
	if got, want := binary.BigEndian.Uint64(frame[1:PTYBinaryOutputFrameHeaderBytes]), offset; got != want {
		t.Fatalf("frame offset = %d, want %d", got, want)
	}
	decodedOffset, decodedOutput, err := DecodePTYBinaryOutputFrame(frame)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	if decodedOffset != offset {
		t.Fatalf("decoded offset = %d, want %d", decodedOffset, offset)
	}
	if !bytes.Equal(decodedOutput, output) {
		t.Fatalf("decoded output = %q, want %q", decodedOutput, output)
	}
}

func TestPTYBinaryOutputFrameEmptyOutput(t *testing.T) {
	frame := EncodePTYBinaryOutputFrame(42, nil)
	if got, want := len(frame), PTYBinaryOutputFrameHeaderBytes; got != want {
		t.Fatalf("frame len = %d, want %d", got, want)
	}
	offset, output, err := DecodePTYBinaryOutputFrame(frame)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	if offset != 42 {
		t.Fatalf("offset = %d, want 42", offset)
	}
	if len(output) != 0 {
		t.Fatalf("output = %q, want empty", output)
	}
}

func TestPTYBinaryOutputFrameLargeOutput(t *testing.T) {
	output := bytes.Repeat([]byte{0xab}, 128*1024)
	frame := EncodePTYBinaryOutputFrame(99, output)
	offset, decoded, err := DecodePTYBinaryOutputFrame(frame)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	if offset != 99 {
		t.Fatalf("offset = %d, want 99", offset)
	}
	if !bytes.Equal(decoded, output) {
		t.Fatalf("decoded output did not match large payload")
	}
}

func TestDecodePTYBinaryOutputFrameRejectsMalformedHeader(t *testing.T) {
	_, _, err := DecodePTYBinaryOutputFrame([]byte{PTYBinaryOutputFrameKind, 0, 1})
	if err == nil {
		t.Fatalf("expected malformed frame error")
	}
	if !strings.Contains(err.Error(), "malformed PTY binary frame") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestDecodePTYBinaryOutputFrameRejectsUnknownKind(t *testing.T) {
	frame := EncodePTYBinaryOutputFrame(0, []byte("x"))
	frame[0] = 0xff
	_, _, err := DecodePTYBinaryOutputFrame(frame)
	if err == nil {
		t.Fatalf("expected unknown frame kind error")
	}
	if !strings.Contains(err.Error(), "unknown PTY binary frame kind 0xff") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestDecodePTYBinaryOutputFrameReturnsOutputCopy(t *testing.T) {
	frame := EncodePTYBinaryOutputFrame(7, []byte("abc"))
	_, output, err := DecodePTYBinaryOutputFrame(frame)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	frame[PTYBinaryOutputFrameHeaderBytes] = 'z'
	if string(output) != "abc" {
		t.Fatalf("decoded output changed after input mutation: %q", output)
	}
	output[0] = 'q'
	_, decodedAgain, err := DecodePTYBinaryOutputFrame(frame)
	if err != nil {
		t.Fatalf("decode frame again: %v", err)
	}
	if string(decodedAgain) != "zbc" {
		t.Fatalf("decoder retained previous output backing array: %q", decodedAgain)
	}
}
