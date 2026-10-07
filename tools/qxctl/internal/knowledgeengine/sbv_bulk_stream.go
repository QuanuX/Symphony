package knowledgeengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// SBVBulkExport retains one checked descriptor. Closing it never removes the
// caller-selected output. The control receipt is small; the file need not be.
type SBVBulkExport struct {
	file   *os.File
	bytes  uint64
	digest string
	suffix []byte
	format string
	info   os.FileInfo
}

type SBVStreamProgress struct {
	SourceBytes     uint64
	OutputBytes     uint64
	CompleteRecords uint64
	Started         bool
	Complete        bool
	RecordBoundary  bool
	OutputFailure   bool
}

// OpenSBVBundleExport binds a no-follow regular file to an already admitted
// native export receipt, with no whole-file allocation or legacy artifact cap.
func OpenSBVBundleExport(request map[string]any, raw json.RawMessage) (*SBVBulkExport, error) {
	if err := ValidateSBVBundleResult("bundle_export", request, raw); err != nil {
		return nil, err
	}
	m, err := sqavObject(raw, maxResponseBytes)
	if err != nil || m["status"] != "complete" {
		return nil, fmt.Errorf("SBV bundle export is not complete")
	}
	path := m["output_path"].(string)
	file, err := openRelativeNoFollow("/", strings.Split(strings.TrimPrefix(path, "/"), "/"))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	size, ok := sbvBundleUint(m["bytes"])
	if err != nil || !ok || !info.Mode().IsRegular() || info.Size() < 0 || uint64(info.Size()) != size {
		_ = file.Close()
		return nil, fmt.Errorf("SBV bundle export file correspondence mismatch")
	}
	suffix := []byte(m["completion_suffix"].(string))
	if uint64(len(suffix)) >= size {
		_ = file.Close()
		return nil, fmt.Errorf("SBV bundle export completion suffix exceeds body")
	}
	return &SBVBulkExport{file: file, bytes: size, digest: m["file_sha256"].(string), suffix: suffix,
		format: m["format"].(string), info: info}, nil
}

func (x *SBVBulkExport) Close() error { return x.file.Close() }

// Stream cooperatively checks cancellation between I/O calls. An arbitrary
// blocking io.Writer cannot be safely preempted; the CLI separately configures
// supported os.File write deadlines. No background write survives this method.
func (x *SBVBulkExport) Stream(ctx context.Context, dst io.Writer, mode string) (progress SBVStreamProgress, err error) {
	if (mode != "json" && mode != "text" && mode != "ndjson") ||
		(mode == "ndjson") != (x.format == "ndjson") {
		return progress, fmt.Errorf("SBV bundle export format mismatch")
	}
	if err = ctx.Err(); err != nil {
		return progress, err
	}
	if _, err = x.file.Seek(0, io.SeekStart); err != nil {
		return progress, err
	}
	hash := sha256.New()
	write := func(p []byte) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		n, e := dst.Write(p)
		if n < 0 || n > len(p) {
			progress.OutputFailure = true
			return fmt.Errorf("SBV output writer returned invalid count")
		}
		progress.Started = progress.Started || n != 0
		if uint64(n) > ^uint64(0)-progress.OutputBytes {
			return fmt.Errorf("SBV output byte count exceeds representation")
		}
		progress.OutputBytes += uint64(n)
		if mode == "ndjson" {
			if n > 0 {
				progress.RecordBoundary = p[n-1] == '\n'
			}
			for _, b := range p[:n] {
				if b == '\n' {
					if progress.CompleteRecords == ^uint64(0) {
						return fmt.Errorf("SBV output record count exceeds representation")
					}
					progress.CompleteRecords++
				}
			}
		}
		if e != nil {
			progress.OutputFailure = true
			return e
		}
		if n != len(p) {
			progress.OutputFailure = true
			return io.ErrShortWrite
		}
		return nil
	}
	text := sbvSafeBulkText{write: write}
	emit := write
	if mode == "text" {
		if err = write([]byte("SBV selected data stream\n")); err != nil {
			return progress, err
		}
		emit = text.push
	}
	buffer := make([]byte, 64*1024)
	remaining := x.bytes - uint64(len(x.suffix))
	for remaining > 0 {
		if err = ctx.Err(); err != nil {
			return progress, err
		}
		count := uint64(len(buffer))
		if remaining < count {
			count = remaining
		}
		n, e := io.ReadFull(x.file, buffer[:int(count)])
		_, _ = hash.Write(buffer[:n])
		progress.SourceBytes += uint64(n)
		if n != 0 {
			if err = emit(buffer[:n]); err != nil {
				return progress, err
			}
		}
		if e != nil {
			return progress, e
		}
		remaining -= uint64(n)
	}
	tail := make([]byte, len(x.suffix))
	n, err := io.ReadFull(x.file, tail)
	_, _ = hash.Write(tail[:n])
	progress.SourceBytes += uint64(n)
	if err != nil {
		return progress, err
	}
	var extra [1]byte
	n, e := x.file.Read(extra[:])
	if n != 0 || !errors.Is(e, io.EOF) {
		return progress, fmt.Errorf("SBV bundle export length changed")
	}
	after, e := x.file.Stat()
	if e != nil {
		return progress, e
	}
	if progress.SourceBytes != x.bytes || hex.EncodeToString(hash.Sum(nil)) != x.digest ||
		string(tail) != string(x.suffix) || !os.SameFile(x.info, after) ||
		after.Size() != x.info.Size() || !after.ModTime().Equal(x.info.ModTime()) {
		return progress, fmt.Errorf("SBV bundle export integrity mismatch")
	}
	// No root-closing brace or complete NDJSON record has escaped yet.
	if err = ctx.Err(); err != nil {
		return progress, err
	}
	if err = emit(tail); err != nil {
		return progress, err
	}
	if mode == "text" && len(text.pending) != 0 {
		return progress, fmt.Errorf("SBV bundle export has incomplete UTF-8")
	}
	progress.Complete = true
	return progress, nil
}

// Only a UTF-8 prefix of at most three bytes survives a call. Data-provided
// terminal control characters never become executable terminal instructions.
type sbvSafeBulkText struct {
	pending []byte
	write   func([]byte) error
}

func (s *sbvSafeBulkText) push(raw []byte) error {
	input := raw
	if len(s.pending) != 0 {
		input = make([]byte, len(s.pending)+len(raw))
		copy(input, s.pending)
		copy(input[len(s.pending):], raw)
		s.pending = s.pending[:0]
	}
	out := make([]byte, 0, 64*1024)
	flush := func() error {
		if len(out) == 0 {
			return nil
		}
		if err := s.write(out); err != nil {
			return err
		}
		out = out[:0]
		return nil
	}
	for len(input) != 0 {
		if !utf8.FullRune(input) {
			s.pending = append(s.pending[:0], input...)
			break
		}
		r, n := utf8.DecodeRune(input)
		if r == utf8.RuneError && n == 1 {
			return fmt.Errorf("SBV bundle export has invalid UTF-8")
		}
		input = input[n:]
		if r < 32 && r != '\n' {
			return fmt.Errorf("SBV bundle JSON contains raw control byte")
		}
		if r >= 127 {
			if r > 65535 {
				a, b := utf16.EncodeRune(r)
				out = fmt.Appendf(out, "\\u%04x\\u%04x", a, b)
			} else {
				out = fmt.Appendf(out, "\\u%04x", r)
			}
		} else {
			out = append(out, byte(r))
		}
		if len(out) >= 64*1024-16 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}
