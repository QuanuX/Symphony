package stavprotocol

import (
	"bytes"
	"errors"
	"testing"
)

func TestCanonicalize(t *testing.T) {
	in := []byte(" { \"z\" : \"line\\n\", \"a\" : 9007199254740991, \"b\" : true } ")
	want := []byte(`{"a":9007199254740991,"b":true,"z":"line\n"}`)
	got, err := Canonicalize(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("canonical bytes\n got: %s\nwant: %s", got, want)
	}
}

func TestCanonicalizeUTF16PropertyOrder(t *testing.T) {
	in := []byte(`{"€":"euro","\r":"cr","דּ":"hebrew","1":"one","😀":"emoji","":"control","ö":"o"}`)
	want := []byte("{\"\\r\":\"cr\",\"1\":\"one\",\"\u0080\":\"control\",\"ö\":\"o\",\"€\":\"euro\",\"😀\":\"emoji\",\"דּ\":\"hebrew\"}")
	got, err := Canonicalize(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("UTF-16 order\n got: %q\nwant: %q", got, want)
	}
}

func TestCanonicalizeRejectsStrictProfileViolations(t *testing.T) {
	tests := map[string][]byte{
		"invalid utf8":       {'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'},
		"duplicate":          []byte(`{"x":1,"x":1}`),
		"null":               []byte(`{"x":null}`),
		"float":              []byte(`{"x":1.0}`),
		"exponent":           []byte(`{"x":1e0}`),
		"negative":           []byte(`{"x":-1}`),
		"negative zero":      []byte(`{"x":-0}`),
		"leading zero":       []byte(`{"x":01}`),
		"unsafe integer":     []byte(`{"x":9007199254740992}`),
		"unpaired surrogate": []byte(`{"x":"\ud800"}`),
		"noncharacter":       []byte(`{"x":"\uffff"}`),
		"trailing":           []byte(`{} {}`),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Canonicalize(input); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestTypedDecodeRequiresCanonicalBytes(t *testing.T) {
	_, err := DecodeQuery([]byte(`{ "after_sequence":0,"event_classes":[],"limit":100,"outcomes":[],"schema":"symphony.stav.query.v1","tops_id":"3f6f2a0e-44fb-4b08-8e84-d0f8f3e1de34"}`))
	if !errors.Is(err, ErrNonCanonicalJSON) {
		t.Fatalf("got %v, want ErrNonCanonicalJSON", err)
	}
}

func TestTypedDecodeRejectsCaseFoldedMember(t *testing.T) {
	_, err := DecodeQuery([]byte(`{"After_sequence":0,"event_classes":[],"limit":100,"outcomes":[],"schema":"symphony.stav.query.v1","tops_id":"3f6f2a0e-44fb-4b08-8e84-d0f8f3e1de34"}`))
	if err == nil {
		t.Fatal("expected exact-name rejection")
	}
}

func TestCanonicalizeRejectsExcessiveDepth(t *testing.T) {
	input := bytes.Repeat([]byte{'['}, maxJSONDepth+1)
	input = append(input, '0')
	input = append(input, bytes.Repeat([]byte{']'}, maxJSONDepth+1)...)
	if _, err := Canonicalize(input); err == nil {
		t.Fatal("expected depth rejection")
	}
}

func FuzzCanonicalizeDoesNotPanic(f *testing.F) {
	f.Add([]byte(`{"a":1}`))
	f.Add([]byte(`{"a":null}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = Canonicalize(input)
	})
}
