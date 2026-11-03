// Package retainedtest copies quiescent synthetic stores for restore regressions.
// It is test support, not an operator backup utility or a concurrent snapshot API.
package retainedtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func Inventory(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			result[rel+"/"] = info.Mode().Perm().String()
			return nil
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			t.Fatalf("not a private regular fixture file: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func Clone(t *testing.T, root string) string {
	t.Helper()
	before := Inventory(t, root)
	target, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dst, 0700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("non-regular fixture entry: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	Equal(t, before, Inventory(t, root))
	Equal(t, before, Inventory(t, target))
	return target
}

func Equal(t *testing.T, expected, actual map[string]string) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatal("retained snapshot bytes or directory modes changed")
	}
}
