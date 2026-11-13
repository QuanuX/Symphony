// Shared release-only fixture support. Build with exactly one qualify-*.go entry.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"time"
	"unicode/utf8"
)

type qObject = map[string]any

func qneed(ok bool, message string) {
	if !ok {
		panic(fmt.Errorf("%s", message))
	}
}
func qerr(err error) {
	if err != nil {
		panic(err)
	}
}
func qmain(fn func()) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, "Qualification failed; retain work/logs:", e)
			os.Exit(1)
		}
	}()
	fn()
}
func qbytes(path string) []byte { b, e := os.ReadFile(path); qerr(e); return b }
func qdigest(path string) string {
	f, e := os.Open(path)
	qerr(e)
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	qerr(e)
	return hex.EncodeToString(h.Sum(nil))
}
func qhash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func qread(path string) qObject {
	raw := qbytes(path)
	qneed(utf8.Valid(raw), "JSON must be valid UTF-8")
	var value qObject
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	qerr(decoder.Decode(&value))
	qneed(value != nil, "JSON object required")
	var extra any
	qneed(decoder.Decode(&extra) == io.EOF, "trailing JSON")
	return value
}
func qo(value any) qObject {
	v, ok := value.(map[string]any)
	qneed(ok, "JSON object required")
	return v
}
func qs(value any) string { v, ok := value.(string); qneed(ok, "JSON string required"); return v }
func qb(value any) bool   { v, ok := value.(bool); qneed(ok, "JSON boolean required"); return v }
func qa(value any) []any  { v, ok := value.([]any); qneed(ok, "JSON array required"); return v }
func qwrite(path string, value any) {
	var b []byte
	if raw, ok := value.([]byte); ok {
		b = raw
	} else {
		var e error
		b, e = json.MarshalIndent(value, "", "  ")
		qerr(e)
		b = append(b, '\n')
	}
	qwriteRaw(path, b)
}
func qwriteRaw(path string, b []byte) {
	qerr(os.MkdirAll(filepath.Dir(path), 0700))
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	qerr(e)
	_, e = f.Write(b)
	closeErr := f.Close()
	qerr(e)
	qerr(closeErr)
	qerr(os.Chmod(path, 0600))
}
func qcopy(source, target string, mode os.FileMode, replace bool) {
	f, e := os.Open(source)
	qerr(e)
	defer f.Close()
	qerr(os.MkdirAll(filepath.Dir(target), 0700))
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if replace {
		flags = os.O_WRONLY | os.O_TRUNC
	}
	out, e := os.OpenFile(target, flags, mode)
	qerr(e)
	_, e = io.Copy(out, f)
	closeErr := out.Close()
	qerr(e)
	qerr(closeErr)
	qerr(os.Chmod(target, mode))
}
func qreplace(path string, b []byte) {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600)
	qerr(e)
	_, e = f.Write(b)
	closeErr := f.Close()
	qerr(e)
	qerr(closeErr)
}
func qcopytree(source, target string) {
	qneed(!qexists(target), "copy destination must be new")
	qerr(filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, e := filepath.Rel(source, path)
		if e != nil {
			return e
		}
		dest := filepath.Join(target, relative)
		if d.IsDir() {
			return os.Mkdir(dest, 0700)
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular fixture input: %s", path)
		}
		qcopy(path, dest, info.Mode().Perm(), false)
		return nil
	}))
}
func qexists(path string) bool {
	_, e := os.Lstat(path)
	if os.IsNotExist(e) {
		return false
	}
	qerr(e)
	return true
}
func qresolve(path string) string {
	p, e := filepath.Abs(path)
	qerr(e)
	p, e = filepath.EvalSymlinks(p)
	qerr(e)
	return p
}
func qwithin(root, path string) bool {
	relative, e := filepath.Rel(root, path)
	qerr(e)
	return relative == "." || relative != ".." && !filepath.IsAbs(relative) && len(relative) > 0 && !(len(relative) > 3 && relative[:3] == ".."+string(os.PathSeparator))
}
func qnewwork(path string, outside ...string) string {
	qneed(filepath.IsAbs(path) && !qexists(path), "new absolute work directory required")
	parent := qresolve(filepath.Dir(path))
	work := filepath.Join(parent, filepath.Base(path))
	for _, root := range outside {
		qneed(!qwithin(qresolve(root), work), "work must be outside selected inputs")
	}
	qerr(os.Mkdir(work, 0700))
	return work
}
func qjoin(base string, parts ...string) string {
	return filepath.Join(append([]string{base}, parts...)...)
}
func qappend(base []string, more ...string) []string {
	return append(append([]string(nil), base...), more...)
}
func qproof(p qObject) {
	qneed(qb(p["selected_artifact_owner_verified"]) && !qb(p["fresh_host_verified"]) && !qb(p["release_acceptance_complete"]) && !qb(p["publication_authorized"]), "overclaimed owner result")
}
func qequal(a, b any) bool { return reflect.DeepEqual(a, b) }
func qfiles(root string) []string {
	var names []string
	qerr(filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if !d.Type().IsRegular() {
				return fmt.Errorf("unexpected nonregular retained entry")
			}
			relative, e := filepath.Rel(root, path)
			if e != nil {
				return e
			}
			names = append(names, filepath.ToSlash(relative))
		}
		return nil
	}))
	sort.Strings(names)
	return names
}

type qRunner struct {
	logs     string
	env      []string
	timeout  time.Duration
	commands []qObject
}

func (r *qRunner) run(label string, args []string, expected int) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = r.env
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	code := 0
	if err != nil {
		if x, ok := err.(*exec.ExitError); ok {
			code = x.ExitCode()
		} else {
			panic(err)
		}
	}
	qneed(out.Len()+errout.Len() <= 16<<20, "unexpected output volume")
	log := qjoin(r.logs, label+".log")
	qwriteRaw(log, append(append([]byte{}, out.Bytes()...), errout.Bytes()...))
	r.commands = append(r.commands, qObject{"label": label, "arguments": args, "exit_code": code, "log_sha256": qdigest(log)})
	qerr(ctx.Err())
	qneed(code == expected, "qualification step failed: "+label)
	return out.Bytes()
}
func qnewRunner(work string, timeout time.Duration, env []string) *qRunner {
	logs := qjoin(work, "logs")
	qerr(os.Mkdir(logs, 0700))
	return &qRunner{logs: logs, env: env, timeout: timeout}
}
func qfixtureKey(work string, r *qRunner) (key string, cleanup func()) {
	dir, e := os.MkdirTemp(work, "fixture-key-")
	qerr(e)
	cleanup = func() { qerr(os.RemoveAll(dir)) }
	key = qjoin(dir, "key")
	return key, cleanup
}
