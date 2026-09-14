// Package shvjob owns private materialization checkpoints, not catalogue heads.
package shvjob

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"path/filepath"
	"regexp"
)

const maxStoreBytes = 1 << 20

var token = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type Store struct{ Root, JobID string }
type Transaction struct {
	raw   json.RawMessage
	write func([]byte, func() error) error
}

func seal(v any) (string, error) { return knowledgeengine.SCVDigest(v) }
func New(root, id string) (Store, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" || !token.MatchString(id) {
		return Store{}, fmt.Errorf("job requires clean absolute root and stable job ID")
	}
	return Store{root, id}, nil
}
func valid(raw []byte) error {
	if e := knowledgeengine.ValidateSCVBundleText(raw); e != nil {
		return e
	}
	var m map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e := d.Decode(&m); e != nil {
		return e
	}
	saved := m["digest"]
	delete(m, "digest")
	want, e := seal(m)
	if e != nil || saved != want {
		return fmt.Errorf("checkpoint seal differs")
	}
	m["digest"] = saved
	canonical, e := knowledgeengine.SCVCanonical(m)
	if e != nil || !bytes.Equal(raw, canonical) {
		return fmt.Errorf("checkpoint must be canonical")
	}
	return nil
}
func (s Store) WithLock(f func(*Transaction) error) error {
	if _, e := New(s.Root, s.JobID); e != nil {
		return e
	}
	return s.withDirectory(func(read func() ([]byte, error), write func([]byte, func() error) error) error {
		raw, e := read()
		if e != nil {
			return e
		}
		if raw != nil {
			if e = valid(raw); e != nil {
				return e
			}
		}
		return f(&Transaction{raw, write})
	})
}
func (t *Transaction) Read() json.RawMessage { return append(json.RawMessage(nil), t.raw...) }
func (t *Transaction) Save(raw json.RawMessage, guard func() error) error {
	if e := valid(raw); e != nil {
		return e
	}
	if e := t.write(raw, guard); e != nil {
		return e
	}
	t.raw = append(json.RawMessage(nil), raw...)
	return nil
}
