//go:build darwin || linux

package profile

import (
	"errors"
	"fmt"
	c "github.com/QuanuX/Symphony/tools/sov-ssh-adapter/profilewire"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Store struct {
	dir, profiles, packages *directory
	scope                   c.Scope
}

func Root() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/Symphony/sov/transport-profiles"
	}
	return "/var/lib/symphony/sov/transport-profiles"
}
func PackageRoot() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/Symphony/transport-packages"
	}
	return "/opt/symphony/transport-packages"
}
func Open(scope c.Scope) (*Store, error) {
	if os.Geteuid() == 0 || !c.ValidScope(scope) {
		return nil, c.ErrBinding
	}
	d, err := openProtected(filepath.Join(Root(), scope.HostID, scope.TOPSID))
	if err != nil {
		return nil, err
	}
	s := &Store{dir: d, scope: scope}
	ok := false
	defer func() {
		if !ok {
			s.Close()
		}
	}()
	raw, err := d.read("scope.json", c.MaxDocumentBytes)
	var record struct {
		Protocol string  `json:"protocol"`
		Scope    c.Scope `json:"scope"`
	}
	if err != nil || c.Decode(raw, &record) != nil || record.Protocol != "symphony.sov.transport-profile-scope.v1" || record.Scope != scope {
		return nil, errTrust
	}
	s.profiles, err = d.child("profiles")
	if err != nil {
		return nil, err
	}
	s.packages, err = openProtected(PackageRoot())
	if err != nil {
		return nil, err
	}
	ok = true
	return s, nil
}
func (s *Store) Close() { s.dir.close(); s.profiles.close(); s.packages.close() }
func (s *Store) ready() error {
	_, err := s.dir.read("pending.json", c.MaxDocumentBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return errors.New("transport_profile.recovery_required")
}
func (s *Store) current(id string) (c.Snapshot, error) {
	var v c.Snapshot
	if !c.ValidUUID(id) {
		return v, c.ErrBinding
	}
	raw, err := s.profiles.read(id+".json", c.MaxDocumentBytes)
	if err != nil || c.Decode(raw, &v) != nil || c.ValidateSnapshot(v) != nil || v.Profile.Scope != s.scope || v.Profile.ProfileID != id {
		return v, errTrust
	}
	return v, nil
}
func (s *Store) permitted(v c.Snapshot) error {
	if os.Geteuid() <= 0 || v.Profile.State != "enabled" {
		return c.ErrBinding
	}
	p := c.Principal{UID: fmt.Sprint(os.Geteuid()), GID: fmt.Sprint(os.Getegid())}
	for _, allowed := range v.Policy.Permitted {
		if allowed == p {
			return s.verifyTrust(v)
		}
	}
	return c.ErrBinding
}
func (s *Store) Resolve(name string) (c.Snapshot, error) {
	release, err := s.dir.lock(false)
	if err != nil {
		return c.Snapshot{}, err
	}
	defer release()
	if err = s.ready(); err != nil {
		return c.Snapshot{}, err
	}
	names, err := s.profiles.names()
	if err != nil {
		return c.Snapshot{}, err
	}
	var found *c.Snapshot
	for _, n := range names {
		if strings.HasPrefix(n, ".stage-") {
			continue
		}
		if !strings.HasSuffix(n, ".json") {
			return c.Snapshot{}, errTrust
		}
		v, e := s.current(strings.TrimSuffix(n, ".json"))
		if e != nil {
			return v, e
		}
		if v.Profile.Configuration.Name == name {
			if found != nil {
				return v, errTrust
			}
			found = &v
		}
	}
	if found == nil {
		return c.Snapshot{}, c.ErrBinding
	}
	return *found, s.permitted(*found)
}

// With holds the stable shared lock through dispatch-start recording and child
// Start. The caller must never retain the lock for the connection lifetime.
func (s *Store) With(e Endpoint, fn func(string, string, string) error) error {
	if Validate(e) != nil || e.Scope != s.scope || fn == nil {
		return c.ErrBinding
	}
	release, err := s.dir.lock(false)
	if err != nil {
		return err
	}
	defer release()
	if err = s.ready(); err != nil {
		return err
	}
	v, err := s.current(e.Snapshot.Profile.ProfileID)
	if err != nil {
		return err
	}
	if v.Digest != e.Snapshot.Digest {
		return c.ErrBinding
	}
	if err = s.permitted(v); err != nil {
		return err
	}
	paths := []string{}
	for _, ref := range []c.PackageRef{v.Profile.Configuration.Adapter, v.Profile.Configuration.SSH, v.Profile.Configuration.Helper} {
		p, err := s.executable(ref)
		if err != nil {
			return err
		}
		paths = append(paths, p)
	}
	if paths[1] != e.Direct.SSH.Path {
		return c.ErrBinding
	}
	return fn(paths[0], paths[1], paths[2])
}
func (s *Store) executable(ref c.PackageRef) (string, error) {
	d, e := s.packages.child(ref.PackageID)
	if e != nil {
		return "", e
	}
	defer d.close()
	v, e := d.child(ref.Version)
	if e != nil {
		return "", e
	}
	defer v.close()
	raw, e := v.read("manifest.json", c.MaxDocumentBytes)
	var m c.PackageManifest
	if e != nil || c.Decode(raw, &m) != nil || !c.ValidPath(m.Executable) {
		return "", errTrust
	}
	return filepath.Join(PackageRoot(), ref.PackageID, ref.Version, m.Executable), nil
}
func Plan(raw []byte, name string) (Endpoint, error) {
	var in Selection
	if Decode(raw, &in) != nil || in.Protocol != SelectionProtocol || name == "" {
		return Endpoint{}, c.ErrBinding
	}
	s, err := Open(in.Scope)
	if err != nil {
		return Endpoint{}, err
	}
	defer s.Close()
	snapshot, err := s.Resolve(name)
	if err != nil {
		return Endpoint{}, err
	}
	e := Endpoint{EndpointProtocol, in.Direct, in.Scope, in.Application, in.SessionDirectory, in.JournalDirectory, snapshot}
	return e, Validate(e)
}
func safeAbsolute(p string) bool {
	return filepath.IsAbs(p) && filepath.Clean(p) == p && p != "/" && !strings.ContainsAny(p, "%\r\n\x00")
}
