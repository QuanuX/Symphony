//go:build darwin || linux

package knowledgelifecycle

import (
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadExecutionScope opens each component without following symbolic links and
// reads one stable, bounded regular file. The caller retains this value for the
// whole invocation instead of rereading a mutable authorization file.
func ReadExecutionScope(path string) (ExecutionScope, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ExecutionScope{}, err
	}
	if !safeAbsolutePath(absolute) || absolute == "/" {
		return ExecutionScope{}, fmt.Errorf("invalid execution scope path")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return ExecutionScope{}, err
	}
	parts := strings.Split(strings.TrimPrefix(absolute, "/"), "/")
	for i, part := range parts {
		flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
		if i < len(parts)-1 {
			flags |= unix.O_DIRECTORY
		}
		next, openErr := unix.Openat(fd, part, flags, 0)
		unix.Close(fd)
		if openErr != nil {
			return ExecutionScope{}, fmt.Errorf("open execution scope: %w", openErr)
		}
		fd = next
	}
	file := os.NewFile(uintptr(fd), absolute)
	defer file.Close()
	var before, after unix.Stat_t
	if err := unix.Fstat(fd, &before); err != nil {
		return ExecutionScope{}, err
	}
	if before.Mode&unix.S_IFMT != unix.S_IFREG || before.Uid != uint32(os.Geteuid()) || before.Mode&0o022 != 0 || before.Nlink != 1 || before.Size <= 0 || before.Size > maxExecutionScopeBytes {
		return ExecutionScope{}, fmt.Errorf("execution scope must be an owned non-writable-by-others bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxExecutionScopeBytes+1))
	if err != nil {
		return ExecutionScope{}, err
	}
	if err := unix.Fstat(fd, &after); err != nil {
		return ExecutionScope{}, err
	}
	if before.Dev != after.Dev || before.Ino != after.Ino || before.Mode != after.Mode || before.Uid != after.Uid || before.Gid != after.Gid || before.Nlink != after.Nlink || before.Mtim != after.Mtim || before.Ctim != after.Ctim || before.Size != after.Size || int64(len(data)) != before.Size {
		return ExecutionScope{}, fmt.Errorf("execution scope changed while reading")
	}
	return DecodeExecutionScope(data)
}
