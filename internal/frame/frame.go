// Package frame implements the bounded byte framing of dispatch/1: a four-byte
// unsigned big-endian payload length followed by that many bytes.
//
// Empty payloads are rejected at the framing level. Nonempty payloads are opaque:
// this package does not validate JSON, CCJ-1, object shape, or caller authority.
// It supplies no handshake, peer authentication, descriptor handling, daemon, or
// launcher. Callers must provide connection security and payload validation.
//
// Callers must serialize reads and writes in each direction of a shared stream.
// After a framing error, discard the stream: a read may have consumed a prefix,
// and a failed write may have emitted one. No resynchronization is attempted.
package frame

import (
	"encoding/binary"
	"errors"
	"io"
)

// MaxPayloadSize is the inclusive payload limit in bytes, excluding the header.
const MaxPayloadSize = 131072

var (
	// ErrEmptyFrame reports a zero payload length, which framing rejects even
	// though this package does not otherwise validate payload content.
	ErrEmptyFrame = errors.New("frame: empty payload")
	// ErrFrameTooLarge reports a payload length above MaxPayloadSize.
	ErrFrameTooLarge = errors.New("frame: payload too large")
)

// Read returns one complete payload, leaving any subsequent frame unread.
// It checks the declared size before allocating or reading the payload.
// On any error it returns a nil payload. Header and payload read errors retain
// io.ReadFull semantics, including io.EOF when that part has no bytes and
// io.ErrUnexpectedEOF when that part is truncated after at least one byte.
func Read(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return nil, ErrEmptyFrame
	}
	if size > MaxPayloadSize {
		return nil, ErrFrameTooLarge
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// Write emits one frame without modifying payload. Invalid lengths are rejected
// before any output. A short write with no error becomes io.ErrShortWrite; other
// writer errors are returned unchanged, even if accompanied by a full count.
// An error may leave a partial frame on the stream; callers must not retry it
// on that stream. Success means the writer accepted the bytes, not durability
// or remote validation of a request.
func Write(w io.Writer, payload []byte) error {
	if len(payload) == 0 {
		return ErrEmptyFrame
	}
	if len(payload) > MaxPayloadSize {
		return ErrFrameTooLarge
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writePart(w, header[:]); err != nil {
		return err
	}
	return writePart(w, payload)
}

func writePart(w io.Writer, p []byte) error {
	n, err := w.Write(p)
	if err != nil {
		return err
	}
	if n != len(p) {
		return io.ErrShortWrite
	}
	return nil
}
