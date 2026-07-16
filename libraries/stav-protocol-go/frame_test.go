package stavprotocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var wire bytes.Buffer
	payload := []byte(`{"schema":"symphony.stav.query.v1"}`)
	if err := WriteFrame(&wire, payload, MaxRequestBytes); err != nil {
		t.Fatal(err)
	}
	if got := binary.BigEndian.Uint32(wire.Bytes()[:4]); got != uint32(len(payload)) {
		t.Fatalf("header length %d", got)
	}
	got, err := ReadFrame(&wire, MaxRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload %q", got)
	}
}

func TestReadFrameRejectsBeforeAllocation(t *testing.T) {
	var wire bytes.Buffer
	_ = binary.Write(&wire, binary.BigEndian, uint32(MaxRequestBytes+1))
	if _, err := ReadFrame(&wire, MaxRequestBytes); !errors.Is(err, ErrFrameSize) {
		t.Fatalf("got %v, want ErrFrameSize", err)
	}
}

func TestReadFrameRejectsPartialPayload(t *testing.T) {
	var wire bytes.Buffer
	_ = binary.Write(&wire, binary.BigEndian, uint32(4))
	wire.Write([]byte("ab"))
	if _, err := ReadFrame(&wire, MaxRequestBytes); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want wrapped io.ErrUnexpectedEOF", err)
	}
}

func TestWriteFrameRejectsEmpty(t *testing.T) {
	if err := WriteFrame(io.Discard, nil, MaxRequestBytes); !errors.Is(err, ErrFrameSize) {
		t.Fatalf("got %v, want ErrFrameSize", err)
	}
}

func FuzzReadFrameDoesNotPanic(f *testing.F) {
	f.Add([]byte{0, 0, 0, 1, 'x'})
	f.Add([]byte{0, 1})
	f.Fuzz(func(t *testing.T, wire []byte) {
		_, _ = ReadFrame(bytes.NewReader(wire), MaxRequestBytes)
	})
}
