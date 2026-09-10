package scvtransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublicTransportRejectsCredentialsAndNonpublicTargets(t *testing.T) {
	for _, value := range []string{"http://example.com/", "https://u:p@example.com/", "https://example.com:8443/", "https://127.0.0.1/", "https://[::ffff:127.0.0.1]/"} {
		if _, err := ValidateURL(value); err == nil {
			t.Fatalf("unsafe URI accepted: %s", value)
		}
	}
	for _, value := range []string{"10.0.0.1", "172.16.1.1", "192.168.1.1", "169.254.169.254", "100.64.1.1", "0.0.0.1", "::1", "fe80::1", "fc00::1", "64:ff9b::a00:1"} {
		if PublicAddress(netip.MustParseAddr(value)) {
			t.Fatalf("nonpublic address accepted: %s", value)
		}
	}
	if !PublicAddress(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("public address rejected")
	}
}

func TestPublicTransportPreservesVersionQueryAndSourceFragment(t *testing.T) {
	source := json.RawMessage(`{"locators":[{"locator_id":"docs","uri":"https://example.com/docs?view=v1#section"}]}`)
	requests := []string{}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.String())
		header := http.Header{}
		code := 200
		if len(requests) == 1 {
			code = 302
			header.Set("Location", "https://example.com/moved?view=v2#new")
		}
		return &http.Response{StatusCode: code, Header: header, Body: io.NopCloser(strings.NewReader("text")), Request: request}, nil
	})
	result, err := acquire(context.Background(), Input{Source: source, LocatorID: "docs"}, transport)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0] != "https://example.com/docs?view=v1" || requests[1] != "https://example.com/moved?view=v2" || result.ResolvedURI != requests[1] || len(result.Redirects) != 1 || string(result.Source) != string(source) {
		t.Fatalf("source/query/redirect drift: %#v %#v", requests, result)
	}
}

func TestPublicTransportBoundsAndQualifiesCaptures(t *testing.T) {
	source := json.RawMessage(`{"locators":[{"locator_id":"docs","uri":"https://example.com/docs"}]}`)
	for _, test := range []struct {
		name string
		code int
		body string
		want string
	}{{"complete", 200, "text", "complete"}, {"bounded", 200, strings.Repeat("a", MaxBytes+1), "partial"}, {"upstream-partial", 206, "fragment", "partial"}, {"http-failure", 503, "unavailable", "failed"}, {"binary", 200, "\x00bad", "failed"}} {
		t.Run(test.name, func(t *testing.T) {
			transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" || request.Method != "GET" {
					t.Fatal("credential or method drift")
				}
				return &http.Response{StatusCode: test.code, Header: http.Header{"Content-Type": []string{"text/plain"}, "Etag": []string{"\"exact-revision\""}}, Body: io.NopCloser(strings.NewReader(test.body)), Request: request}, nil
			})
			result, err := acquire(context.Background(), Input{Source: source, LocatorID: "docs"}, transport)
			if err != nil {
				t.Fatal(err)
			}
			if result.Completeness != test.want || len(result.Body) > MaxBytes || string(result.Source) != string(source) {
				t.Fatalf("capture drift: %#v", result)
			}
			if test.want != "complete" && len(result.Issues) == 0 {
				t.Fatal("incomplete capture lacked explicit issue")
			}
		})
	}
}

func TestPublicTransportDoesNotFollowRejectedRedirect(t *testing.T) {
	calls := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://127.0.0.1/secret"}}, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	})
	result, err := acquire(context.Background(), Input{Source: json.RawMessage(`{"locators":[{"locator_id":"x","uri":"https://example.com/"}]}`), LocatorID: "x"}, transport)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result.Completeness != "failed" || len(result.Redirects) != 0 {
		t.Fatalf("unsafe redirect followed: %d %#v", calls, result)
	}
}

func TestPublicDialFallsBackAcrossPinnedAddressFamilies(t *testing.T) {
	lookups := 0
	attempts := []string{}
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		lookups++
		return []netip.Addr{
			netip.MustParseAddr("2606:4700::1111"), netip.MustParseAddr("2606:4700::1001"), netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.0.0.1")}, nil
	}
	client, server := net.Pipe()
	defer server.Close()
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		attempts = append(attempts, address)
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 2*time.Second {
			t.Fatal("address attempt lacks bounded deadline")
		}
		if len(attempts) == 1 {
			return nil, errors.New("first IPv6 route unavailable")
		}
		return client, nil
	}
	connection, err := publicDialWith(context.Background(), "tcp", "example.com:443", lookup, dial)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if lookups != 1 || len(attempts) != 2 || attempts[0] != "[2606:4700::1111]:443" || attempts[1] != "1.1.1.1:443" {
		t.Fatalf("fallback lost pinned/interleaved selection: lookups=%d attempts=%v", lookups, attempts)
	}
}

func TestPublicDialRejectsAnyNonpublicAddressBeforeDial(t *testing.T) {
	attempts := 0
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("127.0.0.1")}, nil
	}
	dial := func(context.Context, string, string) (net.Conn, error) {
		attempts++
		return nil, errors.New("must not dial")
	}
	_, err := publicDialWith(context.Background(), "tcp", "example.com:443", lookup, dial)
	if err == nil || attempts != 0 || failureIssue(err) != "https_nonpublic_address_rejected" {
		t.Fatalf("unsafe DNS set connected: attempts=%d err=%v", attempts, err)
	}
}

func TestPublicDialStopsAtWholeRetrievalDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempts := 0
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.0.0.1")}, nil
	}
	dial := func(context.Context, string, string) (net.Conn, error) {
		attempts++
		cancel()
		return nil, errors.New("route failed")
	}
	_, err := publicDialWith(ctx, "tcp", "example.com:443", lookup, dial)
	if attempts != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("fallback ignored retrieval cancellation: attempts=%d err=%v", attempts, err)
	}
}

func TestPublicTransportReportsBoundedFailureStage(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, failStage("https_dns_lookup_failed", errors.New("https://example.com/?private-query=value"))
	})
	result, err := acquire(context.Background(), Input{Source: json.RawMessage(`{"locators":[{"locator_id":"x","uri":"https://example.com/?view=v1"}]}`), LocatorID: "x"}, transport)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Issues) != 1 || result.Issues[0] != "https_dns_lookup_failed" {
		t.Fatalf("diagnostic includes unbounded error detail: %#v", result.Issues)
	}
}
