package auditclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	a "github.com/QuanuX/Symphony/tools/sov-transport-profile-admin/internal/transportaudit"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

var unavailable = errors.New("transport_profile.audit_unavailable")

func Socket(tops string) string {
	root := "/run/symphony"
	if runtime.GOOS == "darwin" {
		root = "/private/var/run/symphony"
	}
	return filepath.Join(root, tops, "ssiag", "ssiag.sock")
}
func protectedSocket(path string) error {
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
			return unavailable
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			return unavailable
		}
		if p == "/" {
			break
		}
	}
	info, e := os.Lstat(path)
	if e != nil || info.Mode()&os.ModeSocket == 0 || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0002 != 0 {
		return unavailable
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return unavailable
	}
	return nil
}

// Submit opens only the fixed system SSIAG Unix socket. Peer credentials, not
// an HTTP header, file owner alone or returned digest, authenticate the server.
// No environment proxy, redirect, implicit POST retry or external network path
// is used. A lost response stays uncertain until explicit store recovery.
func Submit(ctx context.Context, input a.Request) (a.Response, error) {
	if input.Validate() != nil {
		return a.Response{}, unavailable
	}
	socket := Socket(input.TOPSID)
	if protectedSocket(socket) != nil {
		return a.Response{}, unavailable
	}
	transport := &http.Transport{DisableKeepAlives: true, MaxResponseHeaderBytes: 16384, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		conn, e := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", socket)
		if e != nil {
			return nil, unavailable
		}
		peer, e := getPeerCredentials(conn)
		if e != nil || peer.UID != 0 {
			conn.Close()
			return nil, unavailable
		}
		return conn, nil
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return unavailable }}
	raw, e := json.Marshal(input)
	if e != nil {
		return a.Response{}, unavailable
	}
	request, e := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost"+a.Endpoint, bytes.NewReader(raw))
	if e != nil {
		return a.Response{}, unavailable
	}
	request.Header.Set("Content-Type", "application/json")
	response, e := client.Do(request)
	if e != nil {
		return a.Response{}, unavailable
	}
	defer response.Body.Close()
	data, e := io.ReadAll(io.LimitReader(response.Body, 65537))
	var result a.Response
	if e != nil || response.StatusCode != 200 || a.Decode(data, &result) != nil || result.Validate(input) != nil {
		return a.Response{}, unavailable
	}
	return result, nil
}
