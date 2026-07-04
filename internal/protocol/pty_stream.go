package protocol

import (
	"encoding/binary"
	"fmt"
)

const (
	PTYBinaryOutputFrameKind        byte = 0x01
	PTYBinaryOutputFrameHeaderBytes      = 9
)

func EncodePTYBinaryOutputFrame(offset uint64, output []byte) []byte {
	frame := make([]byte, PTYBinaryOutputFrameHeaderBytes+len(output))
	frame[0] = PTYBinaryOutputFrameKind
	binary.BigEndian.PutUint64(frame[1:PTYBinaryOutputFrameHeaderBytes], offset)
	copy(frame[PTYBinaryOutputFrameHeaderBytes:], output)
	return frame
}

func DecodePTYBinaryOutputFrame(data []byte) (uint64, []byte, error) {
	if len(data) < PTYBinaryOutputFrameHeaderBytes {
		return 0, nil, fmt.Errorf("malformed PTY binary frame: got %d bytes, want at least %d", len(data), PTYBinaryOutputFrameHeaderBytes)
	}
	if data[0] != PTYBinaryOutputFrameKind {
		return 0, nil, fmt.Errorf("unknown PTY binary frame kind 0x%02x", data[0])
	}
	offset := binary.BigEndian.Uint64(data[1:PTYBinaryOutputFrameHeaderBytes])
	output := append([]byte(nil), data[PTYBinaryOutputFrameHeaderBytes:]...)
	return offset, output, nil
}
