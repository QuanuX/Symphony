package stavprotocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var ErrFrameSize = errors.New("stav: frame size out of range")

// WriteFrame writes one four-byte big-endian length followed by the payload.
// It does not authenticate, authorize, or open a transport.
func WriteFrame(w io.Writer, payload []byte, max uint32) error {
	if len(payload) == 0 || uint64(len(payload)) > uint64(max) {
		return ErrFrameSize
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writeFull(w, header[:]); err != nil {
		return fmt.Errorf("stav: write frame header: %w", err)
	}
	if err := writeFull(w, payload); err != nil {
		return fmt.Errorf("stav: write frame payload: %w", err)
	}
	return nil
}

// ReadFrame validates the announced length before allocating and reads exactly
// one payload. Stream lifecycle and deadlines belong to the caller.
func ReadFrame(r io.Reader, max uint32) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("stav: read frame header: %w", err)
	}
	n := binary.BigEndian.Uint32(header[:])
	if n == 0 || n > max {
		return nil, ErrFrameSize
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("stav: read frame payload: %w", err)
	}
	return payload, nil
}

func writeFull(w io.Writer, b []byte) error {
	for len(b) != 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
