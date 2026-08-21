package outbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
)

const schema = "symphony.accordare.stav-producer.pending.v1"

type Pending struct {
	Candidate       stavprotocol.Candidate `json:"candidate"`
	CandidateDigest string                 `json:"candidate_digest"`
	Schema          string                 `json:"schema"`
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func Open(dir string) (*Store, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("outbox path must be absolute")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("outbox directory is unsafe")
	}
	return &Store{dir: filepath.Clean(dir)}, nil
}

func (store *Store) Put(candidate stavprotocol.Candidate, digest string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := candidate.Validate(); err != nil {
		return err
	}
	computed, err := stavprotocol.CandidateDigest(candidate)
	if err != nil || computed != digest {
		return fmt.Errorf("candidate digest mismatch")
	}
	pending := Pending{Candidate: candidate, CandidateDigest: digest, Schema: schema}
	data, err := json.Marshal(pending)
	if err != nil {
		return err
	}
	path, err := store.path(candidate.Correlation.RequestID)
	if err != nil {
		return err
	}
	if existing, err := read(path); err == nil {
		if !bytes.Equal(existing, data) {
			return fmt.Errorf("outbox request identity conflicts with different evidence")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return writeAtomic(path, data)
}

func (store *Store) Remove(requestID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	path, err := store.path(requestID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncDirectory(store.dir)
}

func (store *Store) List() ([]Pending, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entries, err := os.ReadDir(store.dir)
	if err != nil {
		return nil, err
	}
	if len(entries) > 10_000 {
		return nil, fmt.Errorf("outbox entry bound exceeded")
	}
	result := make([]Pending, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("outbox contains an unsafe entry")
		}
		path := filepath.Join(store.dir, entry.Name())
		data, err := read(path)
		if err != nil {
			return nil, err
		}
		var pending Pending
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&pending); err != nil || decoder.Decode(&struct{}{}) != io.EOF || pending.Schema != schema {
			return nil, fmt.Errorf("outbox entry is invalid")
		}
		computed, err := stavprotocol.CandidateDigest(pending.Candidate)
		if err != nil || computed != pending.CandidateDigest || entry.Name() != pending.Candidate.Correlation.RequestID+".json" {
			return nil, fmt.Errorf("outbox entry binding is invalid")
		}
		result = append(result, pending)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Candidate.Correlation.RequestID < result[j].Candidate.Correlation.RequestID
	})
	return result, nil
}

func (store *Store) Count() (uint64, error) {
	items, err := store.List()
	return uint64(len(items)), err
}

func (store *Store) path(requestID string) (string, error) {
	if err := stavprotocol.ValidateRequestUUID(requestID); err != nil {
		return "", err
	}
	return filepath.Join(store.dir, requestID+".json"), nil
}

func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > stavprotocol.MaxCandidateBytes*2 {
		return nil, fmt.Errorf("outbox entry is unsafe")
	}
	return os.ReadFile(path)
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temporary, err := os.CreateTemp(dir, ".pending-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}
	return syncDirectory(dir)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
