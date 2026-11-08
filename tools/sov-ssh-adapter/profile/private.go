//go:build darwin || linux

package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	c "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profilewire"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func privateDirectory(path string) (*os.File, error) {
	if os.Geteuid() == 0 || !safeAbsolute(path) {
		return nil, c.ErrBinding
	}
	for p := path; ; p = filepath.Dir(p) {
		st, e := os.Lstat(p)
		if e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return nil, c.ErrBinding
		}
		u, ok := st.Sys().(*syscall.Stat_t)
		if !ok || (u.Uid != 0 && int(u.Uid) != os.Geteuid()) {
			return nil, c.ErrBinding
		}
		if p == path {
			if int(u.Uid) != os.Geteuid() || st.Mode().Perm()&0077 != 0 {
				return nil, c.ErrBinding
			}
		} else if st.Mode().Perm()&0022 != 0 && !(u.Uid == 0 && st.Mode()&os.ModeSticky != 0) {
			return nil, c.ErrBinding
		}
		if p == "/" {
			break
		}
	}
	fd, e := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if e != nil {
		return nil, e
	}
	return os.NewFile(uintptr(fd), path), nil
}

type Readiness struct {
	Protocol     string `json:"protocol"`
	Application  string `json:"application"`
	SessionRef   string `json:"session_ref"`
	HelperDigest string `json:"helper_digest"`
	ExpiresAt    string `json:"expires_at"`
}

// This nonsecret private readiness declaration is an input to the independently
// qualified proxy wrapper. It is not proof of provider authentication. The proxy
// must refuse expired/revoked vendor state without initiating a login flow.
func CheckSession(e Endpoint) error {
	d, err := privateDirectory(e.SessionDirectory)
	if err != nil {
		return err
	}
	defer d.Close()
	fd, err := unix.Openat(int(d.Fd()), "readiness.json", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return errors.New("transport_profile.interaction_required")
	}
	f := os.NewFile(uintptr(fd), "readiness.json")
	defer f.Close()
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Uid != uint32(os.Geteuid()) || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0077 != 0 || st.Nlink != 1 {
		return c.ErrBinding
	}
	b, err := io.ReadAll(io.LimitReader(f, 65537))
	var r Readiness
	if err != nil || len(b) > 65536 || Decode(b, &r) != nil {
		return c.ErrBinding
	}
	expires, err := time.Parse(time.RFC3339Nano, r.ExpiresAt)
	if err != nil || !expires.After(time.Now()) || r.Protocol != "symphony.sov.proxy-readiness.v1" || r.Application != e.Application || r.SessionRef != e.Snapshot.Profile.Configuration.SessionRef || r.HelperDigest != e.Snapshot.Profile.Configuration.Helper.ExecutableDigest {
		return errors.New("transport_profile.interaction_required")
	}
	return nil
}

type Journal struct{ file *os.File }

func Begin(e Endpoint, requestID, invocationDigest string) (*Journal, error) {
	if !c.ValidRequestID(requestID) {
		return nil, c.ErrBinding
	}
	d, err := privateDirectory(e.JournalDirectory)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	fd, err := unix.Openat(int(d.Fd()), requestID+".jsonl", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, fmt.Errorf("transport_profile.dispatch_exists_or_unavailable")
	}
	j := &Journal{os.NewFile(uintptr(fd), requestID)}
	if err = j.Record("prepared", map[string]string{"endpoint_digest": Digest(e), "invocation_digest": invocationDigest, "request_id": requestID, "uid": fmt.Sprint(os.Geteuid()), "gid": fmt.Sprint(os.Getegid()), "profile_id": e.Snapshot.Profile.ProfileID, "profile_generation": e.Snapshot.Profile.Generation, "policy_generation": e.Snapshot.Policy.Generation}); err == nil {
		err = d.Sync()
	}
	if err != nil {
		j.Close()
		return nil, err
	}
	return j, nil
}
func (j *Journal) Record(stage string, details any) error {
	b, e := json.Marshal(map[string]any{"protocol": "symphony.sov.transport-dispatch-record.v1", "stage": stage, "details": details})
	if e != nil || len(b) > 65536 {
		return c.ErrBinding
	}
	if _, e = j.file.Write(append(b, '\n')); e != nil {
		return e
	}
	return j.file.Sync()
}
func (j *Journal) Close() {
	if j != nil {
		j.file.Close()
	}
}
