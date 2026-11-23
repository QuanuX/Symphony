package main

import "testing"

func TestDecodeWikiResponse(t *testing.T) {
	const reply = `{"id":1,"result":{"content":[{"type":"text","text":"first"},{"type":"text","text":"second"}]}}`
	for name, data := range map[string]string{
		"json":                 reply,
		"event stream":         ": keep-alive\r\n\r\nevent: message\r\ndata: " + reply + "\r\n\r\n",
		"multiple data fields": "data: {\"id\":1,\ndata: \"result\":{\"content\":[{\"type\":\"text\",\"text\":\"first\\nsecond\"}]}}",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := decodeWikiResponse([]byte(data))
			if err != nil || got != "first\nsecond" {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
	for _, invalid := range []string{
		`{"id":1,"error":{"message":"unavailable"}}`,
		`{"id":1,"result":{"isError":true,"content":[{"type":"text","text":"error"}]}}`,
		`{"id":1,"result":{"content":[]}}`,
		`{"id":2,"result":{"content":[{"type":"text","text":"wrong request"}]}}`,
		"event: message\ndata: truncated",
	} {
		if _, err := decodeWikiResponse([]byte(invalid)); err == nil {
			t.Errorf("accepted invalid response %q", invalid)
		}
	}
}
