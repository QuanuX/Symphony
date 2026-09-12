package knowledgeengine

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReadSCVBundlePackPayload reads only the explicit local packing envelope. The
// native process reader and legacy ReadPayload retain their one-MiB bounds.
// The exact pack schema and independently bounded logical child are checked by
// the caller and codec; the small value allowance accounts for envelope fields.
func ReadSCVBundlePackPayload(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("--input is required for bundle packing")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	requested, err := os.Lstat(abs)
	if err != nil || requested.Mode()&os.ModeSymlink != 0 || !requested.Mode().IsRegular() {
		return nil, fmt.Errorf("bundle packing input must identify a no-follow regular file")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(requested, info) {
		return nil, fmt.Errorf("bundle packing input changed during canonicalization")
	}
	root, relative, err := splitAbsolutePath(resolved)
	if err != nil {
		return nil, err
	}
	data, err := readNoFollowRelative(root, relative, 4<<20)
	if err != nil {
		return nil, err
	}
	if err := ValidateJSONObjectWithValueLimit(data, 4<<20, 32768+256); err != nil {
		return nil, err
	}
	return data, nil
}
