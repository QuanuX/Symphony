package main

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type archiveItem struct {
	name string
	kind byte
	mode int64
	data []byte
	link string
}

func writeArchive(t *testing.T, items []archiveItem, tail []byte) (*os.File, candidate) {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	inv := candidate{}
	for _, item := range items {
		h := &tar.Header{Name: item.name, Typeflag: item.kind, Mode: item.mode, Linkname: item.link}
		if item.kind == tar.TypeReg {
			h.Size = int64(len(item.data))
		}
		if e := w.WriteHeader(h); e != nil {
			t.Fatal(e)
		}
		if item.kind == tar.TypeReg {
			if _, e := w.Write(item.data); e != nil {
				t.Fatal(e)
			}
		}
		inv.Files = append(inv.Files, fileEntry{Path: item.name, SHA256: digest(item.data), Size: int64(len(item.data))})
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	b.Write(tail)
	f, e := os.CreateTemp(t.TempDir(), "archive-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.Close() })
	if _, e = f.Write(b.Bytes()); e != nil {
		t.Fatal(e)
	}
	f.Seek(0, 0)
	inv.Artifact.Size = int64(b.Len())
	inv.Artifact.SHA256 = digest(b.Bytes())
	return f, inv
}
func TestCandidateExtractionExactBytesAndModes(t *testing.T) {
	f, inv := writeArchive(t, []archiveItem{{"lib/data", tar.TypeReg, 0644, []byte("bytes"), ""}, {"bin/cli", tar.TypeReg, 0755, []byte("fixture executable bytes"), ""}}, nil)
	root := filepath.Join(t.TempDir(), "payload")
	if e := extract(f, root, inv); e != nil {
		t.Fatal(e)
	}
	for _, entry := range inv.Files {
		b, e := os.ReadFile(filepath.Join(root, entry.Path))
		if e != nil || digest(b) != entry.SHA256 {
			t.Fatal("bytes", e)
		}
	}
	info, e := os.Stat(filepath.Join(root, "bin/cli"))
	if e != nil || info.Mode().Perm() != 0755 {
		t.Fatal("mode", e)
	}
	f.Seek(0, 0)
	if e = extract(f, root, inv); e == nil {
		t.Fatal("existing prefix overwritten")
	}
}
func TestCandidateExtractionRefusesUnsafeArchives(t *testing.T) {
	for _, name := range []string{"symlink", "hardlink", "device", "traversal", "absolute", "duplicate", "unlisted", "missing", "digest", "size", "setuid", "writable", "archive-hash", "trailing", "parent-file"} {
		t.Run(name, func(t *testing.T) {
			items := []archiveItem{{"lib/data", tar.TypeReg, 0644, []byte("selected bytes"), ""}}
			var tail []byte
			switch name {
			case "symlink":
				items[0].kind = tar.TypeSymlink
				items[0].link = "../../escape"
			case "hardlink":
				items[0].kind = tar.TypeLink
				items[0].link = "../../escape"
			case "device":
				items[0].kind = tar.TypeChar
			case "traversal":
				items[0].name = "../escape"
			case "absolute":
				items[0].name = "/escape"
			case "duplicate":
				items = append(items, items[0])
			case "setuid":
				items[0].mode = 04755
			case "writable":
				items[0].mode = 0666
			case "trailing":
				tail = []byte("unexpected trailing payload")
			case "parent-file":
				items = append([]archiveItem{{"lib", tar.TypeReg, 0644, []byte("file"), ""}}, items...)
			}
			f, inv := writeArchive(t, items, tail)
			switch name {
			case "duplicate":
				inv.Files = inv.Files[:1]
			case "unlisted":
				inv.Files[0].Path = "lib/other"
			case "missing":
				inv.Files = append(inv.Files, fileEntry{Path: "missing", Size: 0, SHA256: digest(nil)})
			case "digest":
				inv.Files[0].SHA256 = digest([]byte("other"))
			case "size":
				inv.Files[0].Size++
			case "archive-hash":
				inv.Artifact.SHA256 = digest(nil)
			}
			parent := t.TempDir()
			root := filepath.Join(parent, "payload")
			if e := extract(f, root, inv); e == nil {
				t.Fatal("unsafe archive accepted")
			}
			if _, e := os.Lstat(filepath.Join(parent, "escape")); !os.IsNotExist(e) {
				t.Fatal("escaped prefix")
			}
		})
	}
}
func TestCandidateExtractionExistingSymlinkPrefixRefused(t *testing.T) {
	f, inv := writeArchive(t, []archiveItem{{"data", tar.TypeReg, 0644, []byte("x"), ""}}, nil)
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "payload")
	if e := os.Symlink(target, link); e != nil {
		t.Fatal(e)
	}
	if e := extract(f, link, inv); e == nil {
		t.Fatal("followed existing prefix symlink")
	}
	entries, e := os.ReadDir(target)
	if e != nil || len(entries) != 0 {
		t.Fatal("changed symlink target")
	}
}
