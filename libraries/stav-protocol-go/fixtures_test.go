package stavprotocol

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fixturePath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "knowledge", "stav", "fixtures", "v1"))
	return filepath.Join(append([]string{root}, parts...)...)
}

func readFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	b, err := os.ReadFile(fixturePath(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(b)
}

func TestValidFixturesRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		decode func([]byte) ([]byte, error)
	}{
		{"candidate.json", func(b []byte) ([]byte, error) {
			v, err := DecodeCandidate(b)
			if err != nil {
				return nil, err
			}
			return EncodeCandidate(v)
		}},
		{"event.json", func(b []byte) ([]byte, error) {
			v, err := DecodeEvent(b)
			if err != nil {
				return nil, err
			}
			return EncodeEvent(v)
		}},
		{"receipt-rejected.json", func(b []byte) ([]byte, error) {
			v, err := DecodeReceipt(b)
			if err != nil {
				return nil, err
			}
			return EncodeReceipt(v)
		}},
		{"query.json", func(b []byte) ([]byte, error) {
			v, err := DecodeQuery(b)
			if err != nil {
				return nil, err
			}
			return EncodeQuery(v)
		}},
		{"query-page.json", func(b []byte) ([]byte, error) {
			v, err := DecodeQueryPage(b)
			if err != nil {
				return nil, err
			}
			return EncodeQueryPage(v)
		}},
		{"verification.json", func(b []byte) ([]byte, error) {
			v, err := DecodeVerification(b)
			if err != nil {
				return nil, err
			}
			return EncodeVerification(v)
		}},
		{"append-authority-config.json", func(b []byte) ([]byte, error) {
			v, err := DecodeAppendAuthorityConfig(b)
			if err != nil {
				return nil, err
			}
			return EncodeAppendAuthorityConfig(v)
		}},
		{"append-authority-status.json", func(b []byte) ([]byte, error) {
			v, err := DecodeAppendAuthorityStatus(b)
			if err != nil {
				return nil, err
			}
			return EncodeAppendAuthorityStatus(v)
		}},
		{"local-request-status.json", func(b []byte) ([]byte, error) {
			v, err := DecodeLocalRequest(b)
			if err != nil {
				return nil, err
			}
			return EncodeLocalRequest(v)
		}},
		{"local-response-status.json", func(b []byte) ([]byte, error) {
			v, err := DecodeLocalResponse(b)
			if err != nil {
				return nil, err
			}
			return EncodeLocalResponse(v)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := readFixture(t, "valid", tt.name)
			got, err := tt.decode(input)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, input) {
				t.Fatalf("round-trip mismatch\n got: %s\nwant: %s", got, input)
			}
		})
	}
}

func TestInvalidFixtures(t *testing.T) {
	canonicalFailures := []string{
		"candidate-duplicate-key.json",
		"candidate-null.json",
		"query-float.json",
		"query-unsafe-integer.json",
	}
	for _, name := range canonicalFailures {
		t.Run(name, func(t *testing.T) {
			if _, err := Canonicalize(readFixture(t, "invalid", name)); err == nil {
				t.Fatal("expected strict JSON rejection")
			}
		})
	}
	t.Run("query-unknown-field.json", func(t *testing.T) {
		if _, err := DecodeQuery(readFixture(t, "invalid", "query-unknown-field.json")); err == nil {
			t.Fatal("expected typed unknown-field rejection")
		}
	})
	t.Run("local-request-multiple-payloads.json", func(t *testing.T) {
		if _, err := DecodeLocalRequest(readFixture(t, "invalid", "local-request-multiple-payloads.json")); err == nil {
			t.Fatal("expected local request union rejection")
		}
	})
	t.Run("local-response-wrong-payload.json", func(t *testing.T) {
		if _, err := DecodeLocalResponse(readFixture(t, "invalid", "local-response-wrong-payload.json")); err == nil {
			t.Fatal("expected local response union rejection")
		}
	})
}

func TestDigestVectors(t *testing.T) {
	candidate, err := DecodeCandidate(readFixture(t, "valid", "candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotCandidate, err := CandidateDigest(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if want := "sha256:c049502292f1e073b29f3db0108af59676cc8ffebf1b22217fd8b64eb2a70629"; gotCandidate != want {
		t.Fatalf("candidate digest %s, want %s", gotCandidate, want)
	}

	event, err := DecodeEvent(readFixture(t, "valid", "event.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotEvent, err := EventDigest(event)
	if err != nil {
		t.Fatal(err)
	}
	if want := "sha256:4236aee922a67725aa5b90e22e88bfcf0aa510875f03777b82e326a1ffa5eef2"; gotEvent != want {
		t.Fatalf("event digest %s, want %s", gotEvent, want)
	}

	gotGenesis, err := GenesisDigest("3f6f2a0e-44fb-4b08-8e84-d0f8f3e1de34")
	if err != nil {
		t.Fatal(err)
	}
	if want := "sha256:702b51f582cbb90a36566ed23e741a2f14d577b469b27cd4b3c9c8c92f6130d7"; gotGenesis != want {
		t.Fatalf("genesis digest %s, want %s", gotGenesis, want)
	}
}
