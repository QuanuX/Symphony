// Package scvtransport collects bounded public HTTPS text. It has no provider
// credentials, source-configuration write path, parser, or authority inference.
package scvtransport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxBytes = 65536
const MaxRedirects = 8

type Input struct {
	Source    json.RawMessage `json:"source"`
	LocatorID string          `json:"locator_id"`
}
type CaptureInput struct {
	Source           json.RawMessage `json:"source"`
	LocatorID        string          `json:"locator_id"`
	ResolvedURI      string          `json:"resolved_uri"`
	Redirects        []string        `json:"redirects"`
	ObservedAt       string          `json:"observed_at"`
	UpstreamRevision any             `json:"upstream_revision"`
	MediaType        string          `json:"media_type"`
	Body             string          `json:"body"`
	Completeness     string          `json:"completeness"`
	Issues           []string        `json:"issues"`
}

func ValidateURL(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.Port() != "" && u.Port() != "443" {
		return nil, fmt.Errorf("public text adapter requires HTTPS on port 443 without userinfo")
	}
	if strings.ContainsAny(value, "\r\n\x00\t ") || len(value) > 4096 {
		return nil, fmt.Errorf("unsafe public HTTPS URI")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !PublicAddress(ip) {
		return nil, fmt.Errorf("HTTPS address is not public")
	}
	return u, nil
}

// Exclude special-use IPv4/IPv6 ranges as well as private/loopback/link-local.
// Dialing uses the validated numeric address, preventing DNS rebinding between
// policy check and connect. Each redirect performs the same independent check.
func PublicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, value := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001::/23", "2001:db8::/32", "2002::/16", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64"} {
		prefix := netip.MustParsePrefix(value)
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := net.Dialer{KeepAlive: -1}
	return publicDialWith(ctx, network, address, net.DefaultResolver.LookupNetIP, dialer.DialContext)
}

type transportFailure struct {
	stage string
	cause error
}

func (failure *transportFailure) Error() string { return failure.stage }
func (failure *transportFailure) Unwrap() error { return failure.cause }
func failStage(stage string, cause error) error { return &transportFailure{stage: stage, cause: cause} }

// Resolve once, validate the complete returned set before any connection, then
// alternate address families while dialing only those pinned numeric addresses.
// A broken first IPv6 route must not consume the entire retrieval deadline when
// the same public origin has a working IPv4 address. Every attempt is bounded
// to two seconds and the caller's earlier whole-retrieval deadline still wins.
func publicDialWith(ctx context.Context, network, address string,
	lookup func(context.Context, string, string) ([]netip.Addr, error),
	dial func(context.Context, string, string) (net.Conn, error)) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" {
		return nil, failStage("https_target_rejected", nil)
	}
	addresses, err := lookup(ctx, "ip", host)
	if err != nil {
		return nil, failStage("https_dns_lookup_failed", err)
	}
	if len(addresses) == 0 || len(addresses) > 32 {
		return nil, failStage("https_dns_address_count_rejected", nil)
	}
	for _, ip := range addresses {
		if !PublicAddress(ip) {
			return nil, failStage("https_nonpublic_address_rejected", nil)
		}
	}
	firstFamilyV6 := addresses[0].Unmap().Is6()
	primary, alternate := []netip.Addr{}, []netip.Addr{}
	seen := map[netip.Addr]bool{}
	for _, ip := range addresses {
		ip = ip.Unmap()
		if seen[ip] {
			continue
		}
		seen[ip] = true
		if ip.Is6() == firstFamilyV6 {
			primary = append(primary, ip)
		} else {
			alternate = append(alternate, ip)
		}
	}
	ordered := make([]netip.Addr, 0, len(seen))
	for i := 0; i < len(primary) || i < len(alternate); i++ {
		if i < len(primary) {
			ordered = append(ordered, primary[i])
		}
		if i < len(alternate) {
			ordered = append(ordered, alternate[i])
		}
	}
	var lastError error
	for _, ip := range ordered {
		if err := ctx.Err(); err != nil {
			return nil, failStage("https_connect_deadline", err)
		}
		attempt, cancel := context.WithTimeout(ctx, 2*time.Second)
		connection, err := dial(attempt, network, net.JoinHostPort(ip.String(), port))
		cancel()
		if err == nil {
			return connection, nil
		}
		if connection != nil {
			_ = connection.Close()
		}
		lastError = err
	}
	return nil, failStage("https_connect_failed", lastError)
}

// These finite stage codes intentionally omit raw network error strings, which
// can contain full request URLs or query parameters supplied by the source.
func failureIssue(err error) string {
	var staged *transportFailure
	if errors.As(err, &staged) {
		return staged.stage
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "https_retrieval_deadline"
	}
	if errors.Is(err, context.Canceled) {
		return "https_retrieval_canceled"
	}
	var certificate *tls.CertificateVerificationError
	if errors.As(err, &certificate) {
		return "https_tls_certificate_rejected"
	}
	return "https_request_failed"
}

func Acquire(ctx context.Context, input Input) (CaptureInput, error) {
	transport := &http.Transport{Proxy: nil, DialContext: publicDial, DisableCompression: true, DisableKeepAlives: true,
		TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, MaxResponseHeaderBytes: 65536}
	defer transport.CloseIdleConnections()
	return acquire(ctx, input, transport)
}

func acquire(ctx context.Context, input Input, transport http.RoundTripper) (CaptureInput, error) {
	var source struct {
		Locators []struct {
			ID  string `json:"locator_id"`
			URI string `json:"uri"`
		} `json:"locators"`
	}
	if json.Unmarshal(input.Source, &source) != nil {
		return CaptureInput{}, fmt.Errorf("invalid selected source")
	}
	uri := ""
	for _, locator := range source.Locators {
		if locator.ID == input.LocatorID {
			if uri != "" {
				return CaptureInput{}, fmt.Errorf("duplicate locator")
			}
			uri = locator.URI
		}
	}
	if uri == "" {
		return CaptureInput{}, fmt.Errorf("locator not present in selected source revision")
	}
	selectedURL, err := ValidateURL(uri)
	if err != nil {
		return CaptureInput{}, err
	}
	// Fragments are local document selectors, never HTTP request components.
	// The source document retains the exact configured URI including fragment.
	selectedURL.Fragment = ""
	selectedURL.RawFragment = ""
	uri = selectedURL.String()
	result := CaptureInput{Source: input.Source, LocatorID: input.LocatorID, ResolvedURI: uri, Redirects: []string{},
		ObservedAt: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), MediaType: "application/octet-stream", Completeness: "failed", Issues: []string{}}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	client := &http.Client{Transport: transport, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) > MaxRedirects {
			return failStage("https_redirect_limit_exceeded", nil)
		}
		if _, err := ValidateURL(request.URL.String()); err != nil {
			return failStage("https_redirect_target_rejected", err)
		}
		request.URL.Fragment = ""
		request.URL.RawFragment = ""
		result.Redirects = append(result.Redirects, request.URL.String())
		result.ResolvedURI = request.URL.String()
		return nil
	}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return CaptureInput{}, err
	}
	request.Header.Set("Accept", "text/plain, text/html, application/json, application/xml, text/markdown")
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("User-Agent", "Symphony-SCV-PublicText/0.1.0-dev")
	response, err := client.Do(request)
	if err != nil {
		result.Issues = append(result.Issues, failureIssue(err))
		return result, nil
	}
	defer response.Body.Close()
	result.ResolvedURI = response.Request.URL.String()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		result.Issues = append(result.Issues, fmt.Sprintf("http_status_%d", response.StatusCode))
	} else if response.StatusCode == http.StatusPartialContent {
		result.Completeness = "partial"
		result.Issues = append(result.Issues, "http_partial_content")
	} else {
		result.Completeness = "complete"
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		result.Completeness = "failed"
		result.Issues = append(result.Issues, "unsupported_content_encoding")
		return result, nil
	}
	if media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type")); err == nil && len(media) <= 128 {
		result.MediaType = media
	}
	for _, header := range []struct{ name, scheme string }{{"ETag", "etag"}, {"Last-Modified", "last-modified"}} {
		value := response.Header.Get(header.name)
		if value != "" && len(value) <= 4096 && !strings.ContainsAny(value, "\r\n\x00") {
			result.UpstreamRevision = map[string]string{"scheme": header.scheme, "value": value}
			break
		}
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, MaxBytes+1))
	if len(body) > MaxBytes {
		body = body[:MaxBytes]
		if result.Completeness != "failed" {
			result.Completeness = "partial"
		}
		result.Issues = append(result.Issues, "body_limit_65536_bytes")
		// Remove only the incomplete last codepoint at a byte-limited boundary.
		for len(body) > 0 && !utf8.Valid(body) && len(body) > MaxBytes-4 {
			body = body[:len(body)-1]
		}
	}
	if readErr != nil {
		if result.Completeness != "failed" {
			result.Completeness = "partial"
		}
		result.Issues = append(result.Issues, "body_read_incomplete")
	}
	if !utf8.Valid(body) || strings.IndexByte(string(body), 0) >= 0 {
		result.Completeness = "failed"
		result.Issues = append(result.Issues, "body_is_not_supported_utf8_text")
		body = nil
	}
	result.Body = string(body)
	return result, nil
}
