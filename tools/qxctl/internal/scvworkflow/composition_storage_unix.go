//go:build darwin || linux

package scvworkflow

import (
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvcorpus"
	"golang.org/x/sys/unix"
)

// WithComposition shares the exact existing owned immutable object store while
// isolating composition journals from the earlier three-stage run protocol.
func (store Store) WithComposition(operation string, create bool, work func(*Session, *CompositionRun) error) error {
	if !scvcorpus.ValidID(operation) {
		return fmt.Errorf("invalid composition operation ID")
	}
	return store.With("", create, func(s *Session, _ *Run) error {
		runs, err := child(s.namespaceDirectory, "composition-runs-v1", true, create)
		if err != nil {
			return err
		}
		defer unix.Close(runs)
		directory, err := child(runs, keyForOperation(operation), true, create)
		if err != nil {
			return err
		}
		defer unix.Close(directory)
		s.runDirectory = directory
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
		if create {
			flags = unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CREAT
		}
		lock, err := unix.Openat(directory, "run.lock", flags, 0o600)
		if err != nil {
			return err
		}
		defer unix.Close(lock)
		if err = privateFile(lock); err != nil {
			return err
		}
		mode := unix.LOCK_SH
		if create {
			mode = unix.LOCK_EX
		}
		if err = unix.Flock(lock, mode|unix.LOCK_NB); err != nil {
			return fmt.Errorf("composition workflow busy: %w", err)
		}
		defer unix.Flock(lock, unix.LOCK_UN)
		raw, err := readFile(directory, "run.json", MaxBytes)
		if err != nil {
			return err
		}
		if raw == nil {
			return work(s, nil)
		}
		run, err := ReadCompositionRun(raw, store, operation)
		if err != nil {
			return err
		}
		return work(s, &run)
	})
}
func (s *Session) SaveComposition(store Store, run CompositionRun) (CompositionRun, error) {
	if s.runDirectory < 0 {
		return run, fmt.Errorf("composition run not locked")
	}
	raw, err := Seal(run)
	if err != nil {
		return run, err
	}
	sealed, err := ReadCompositionRun(raw, store, run.OperationID)
	if err != nil {
		return run, err
	}
	previous, err := readFile(s.runDirectory, "run.json", MaxBytes)
	if err != nil {
		return run, err
	}
	if previous != nil {
		old, err := ReadCompositionRun(previous, store, run.OperationID)
		if err != nil {
			return run, err
		}
		if old.IntentDigest != run.IntentDigest || (old.Complete && !run.Complete) {
			return run, fmt.Errorf("cannot replace pinned composition intent/completion")
		}
		for name, c := range old.Stages {
			n, ok := run.Stages[name]
			if !ok || n.InputRef != c.InputRef || (c.ResultRef != "" && n.ResultRef != c.ResultRef) {
				return run, fmt.Errorf("cannot replace composition checkpoint")
			}
		}
	}
	if err = publish(s.runDirectory, "run.json", raw, false); err != nil {
		return run, err
	}
	return sealed, nil
}
