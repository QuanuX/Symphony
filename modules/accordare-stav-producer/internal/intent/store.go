package intent

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
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	accordareprotocol "github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/protocol"
)

const schema = "symphony.accordare.stav-producer.intent.v1"

type Intent struct {
	IntentID   string                       `json:"intent_id"`
	Peer       stavprotocol.SafeReference   `json:"peer"`
	PreparedAt time.Time                    `json:"prepared_at"`
	Schema     string                       `json:"schema"`
	Submission accordareprotocol.Submission `json:"submission"`
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func Open(dir string) (*Store, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("intent path must be absolute")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("intent directory is unsafe")
	}
	return &Store{dir: filepath.Clean(dir)}, nil
}

func (store *Store) Put(value Intent) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := validate(value); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	path, err := store.path(value.IntentID)
	if err != nil {
		return err
	}
	if existing, readErr := read(path); readErr == nil {
		var prior Intent
		if err := json.Unmarshal(existing, &prior); err != nil || prior.IntentID != value.IntentID || prior.Peer != value.Peer || !accordareprotocol.CommandsMatchIntent(prior.Submission.Command, value.Submission.Command) || prior.Submission.Coordinator != value.Submission.Coordinator || prior.Submission.Operation != value.Submission.Operation || prior.Submission.Schema != value.Submission.Schema || prior.Submission.TOPSID != value.Submission.TOPSID {
			return fmt.Errorf("intent identity conflicts with different evidence")
		}
		return nil
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	return writeAtomic(path, data)
}

func (store *Store) Get(intentID string) (Intent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	path, err := store.path(intentID)
	if err != nil {
		return Intent{}, err
	}
	data, err := read(path)
	if err != nil {
		return Intent{}, err
	}
	var value Intent
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil || decoder.Decode(&struct{}{}) != io.EOF || validate(value) != nil {
		return Intent{}, fmt.Errorf("intent entry is invalid")
	}
	return value, nil
}

func (store *Store) Remove(intentID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	path, err := store.path(intentID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncDirectory(store.dir)
}

func (store *Store) List() ([]Intent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entries, err := os.ReadDir(store.dir)
	if err != nil {
		return nil, err
	}
	if len(entries) > 10_000 {
		return nil, fmt.Errorf("intent entry bound exceeded")
	}
	result := make([]Intent, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("intent store contains an unsafe entry")
		}
		data, err := read(filepath.Join(store.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var value Intent
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&value); err != nil || decoder.Decode(&struct{}{}) != io.EOF || validate(value) != nil || entry.Name() != value.IntentID+".json" {
			return nil, fmt.Errorf("intent entry binding is invalid")
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].IntentID < result[j].IntentID })
	return result, nil
}

func (store *Store) Count() (uint64, error) {
	values, err := store.List()
	return uint64(len(values)), err
}

func validate(value Intent) error {
	if value.Schema != schema || value.PreparedAt.Location() != time.UTC || value.PreparedAt.Nanosecond() != 0 || value.PreparedAt.IsZero() {
		return fmt.Errorf("intent identity is invalid")
	}
	if err := stavprotocol.ValidateRequestUUID(value.IntentID); err != nil {
		return err
	}
	if value.Peer.ID == "" || value.Peer.Kind == "" || !nullResult(value.Submission.Result) || value.Submission.Outcome != nil || value.Submission.ReasonCode != nil {
		return fmt.Errorf("intent evidence is invalid")
	}
	return nil
}

func nullResult(raw json.RawMessage) bool {
	return len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func (store *Store) path(id string) (string, error) {
	if err := stavprotocol.ValidateRequestUUID(id); err != nil {
		return "", err
	}
	return filepath.Join(store.dir, id+".json"), nil
}

func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > 2<<20 {
		return nil, fmt.Errorf("intent entry is unsafe")
	}
	return os.ReadFile(path)
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temporary, err := os.CreateTemp(dir, ".intent-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
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
	if err := os.Rename(name, path); err != nil {
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
