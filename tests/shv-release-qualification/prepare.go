// This test-only downloader selects one reviewed decoder, never a latest release.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type file struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}
type asset struct {
	Platform, Archive, URL, SHA256, Library string
	Bytes                                   int64
	Files                                   []file
	Required                                bool
}
type source struct {
	URL, Publication, Revision, IssueDate, SHA256, Profile string
	Bytes                                                  int64
}
type lock struct {
	Version string
	Assets  []asset
	Source  source
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func writeJSON(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	check(e)
	check(os.WriteFile(p, append(b, '\n'), 0600))
}
func readRegular(p string, limit int64) []byte {
	s, e := os.Lstat(p)
	check(e)
	if !s.Mode().IsRegular() || s.Size() > limit {
		panic("not a bounded regular file: " + p)
	}
	f, e := os.Open(p)
	check(e)
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	check(e)
	if int64(len(b)) > limit {
		panic("file exceeded bound")
	}
	return b
}
func download(url string, size int64) []byte {
	client := http.Client{Timeout: 60 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 5 {
			return fmt.Errorf("unsafe redirect")
		}
		return nil
	}}
	r, e := client.Get(url)
	check(e)
	defer r.Body.Close()
	if r.StatusCode != 200 {
		panic(fmt.Sprintf("download HTTP %d: %s", r.StatusCode, url))
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, size+1))
	check(e)
	if int64(len(b)) != size {
		panic("download size mismatch")
	}
	return b
}
func verify(b []byte, size int64, digest string) {
	if int64(len(b)) != size || hash(b) != digest {
		panic("pinned byte identity mismatch")
	}
}
func extract(b []byte, root string) {
	g, e := gzip.NewReader(bytes.NewReader(b))
	check(e)
	defer g.Close()
	t := tar.NewReader(g)
	for n := 0; ; n++ {
		h, e := t.Next()
		if e == io.EOF {
			break
		}
		check(e)
		if n > 256 {
			panic("archive member bound")
		}
		name := strings.TrimPrefix(h.Name, "./")
		if name == "" || name == "." {
			continue
		}
		clean := filepath.Clean(name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(name, "\\") {
			panic("unsafe archive path")
		}
		p := filepath.Join(root, clean)
		if h.Typeflag == tar.TypeDir {
			check(os.MkdirAll(p, 0700))
			continue
		}
		if h.Typeflag != tar.TypeReg || h.Size < 0 || h.Size > 64<<20 {
			panic("non-regular or oversized archive member")
		}
		check(os.MkdirAll(filepath.Dir(p), 0700))
		f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		check(e)
		_, e = io.CopyN(f, t, h.Size)
		check(e)
		check(f.Close())
	}
}
func main() {
	work := flag.String("work", "", "fresh absolute workspace")
	catalog := flag.String("lock", "", "reviewed dependency lock")
	retained := flag.String("source", "", "optional retained exact AMD PDF")
	archive := flag.String("archive", "", "optional retained exact selected archive")
	flag.Parse()
	if !filepath.IsAbs(*work) || *catalog == "" {
		panic("absolute work and lock required")
	}
	check(os.Mkdir(*work, 0700))
	root, e := filepath.EvalSymlinks(*work)
	check(e)
	raw := readRegular(*catalog, 1<<20)
	var l lock
	check(json.Unmarshal(raw, &l))
	platform := runtime.GOOS + "-" + runtime.GOARCH
	var selected *asset
	for i := range l.Assets {
		if l.Assets[i].Platform == platform {
			selected = &l.Assets[i]
		}
	}
	if selected == nil {
		panic("unqualified platform: " + platform)
	}
	a := *selected
	var b []byte
	if *archive != "" {
		b = readRegular(*archive, a.Bytes)
	} else {
		b = download(a.URL, a.Bytes)
	}
	verify(b, a.Bytes, a.SHA256)
	check(os.WriteFile(filepath.Join(root, a.Archive), b, 0600))
	decoder := filepath.Join(root, "decoder")
	check(os.Mkdir(decoder, 0700))
	extract(b, decoder)
	for _, f := range a.Files {
		verify(readRegular(filepath.Join(decoder, f.Path), f.Bytes), f.Bytes, f.SHA256)
	}
	library := filepath.Join(decoder, a.Library)
	fileType, e := exec.Command("file", library).CombinedOutput()
	check(e)
	arch := map[string]string{"arm64": "arm64", "amd64": "x86_64"}[runtime.GOARCH]
	if runtime.GOOS == "linux" {
		arch = map[string]string{"arm64": "ARM aarch64", "amd64": "x86-64"}[runtime.GOARCH]
	}
	if !strings.Contains(string(fileType), arch) {
		panic("decoder machine architecture mismatch")
	}
	var loader []byte
	if runtime.GOOS == "darwin" {
		loader, e = exec.Command("otool", "-L", library).CombinedOutput()
	} else {
		loader, e = exec.Command("readelf", "-d", library).CombinedOutput()
	}
	check(e)
	// The selected standalone archives have no separate ICU/font data file. Keep
	// every bundled license and reject dependencies outside the platform runtime.
	for _, line := range strings.Split(string(loader), "\n") {
		if runtime.GOOS == "darwin" && strings.HasPrefix(line, "\t") {
			p := strings.Fields(line)[0]
			if p != "./libpdfium.dylib" && !strings.HasPrefix(p, "/System/Library/") && !strings.HasPrefix(p, "/usr/lib/") {
				panic("unexpected decoder dependency: " + p)
			}
		}
		if runtime.GOOS == "linux" && strings.Contains(line, "(NEEDED)") {
			admitted := false
			for _, lib := range []string{"libc.so.6", "libm.so.6", "libdl.so.2", "libpthread.so.0", "libgcc_s.so.1", "libstdc++.so.6", "ld-linux-x86-64.so.2", "ld-linux-aarch64.so.1"} {
				if strings.Contains(line, "["+lib+"]") {
					admitted = true
				}
			}
			if !admitted {
				panic("unexpected decoder dependency: " + line)
			}
		}
	}
	check(os.WriteFile(filepath.Join(root, "decoder-loader.log"), loader, 0600))
	check(os.WriteFile(filepath.Join(root, "decoder-machine.log"), fileType, 0600))
	acquisition := "pinned_https_download"
	if *retained != "" {
		b = readRegular(*retained, l.Source.Bytes)
		acquisition = "explicit_retained_file"
	} else {
		b = download(l.Source.URL, l.Source.Bytes)
	}
	verify(b, l.Source.Bytes, l.Source.SHA256)
	sources := filepath.Join(root, "sources")
	check(os.Mkdir(sources, 0700))
	check(os.WriteFile(filepath.Join(sources, "amd-69290.pdf"), b, 0600))
	libBytes := readRegular(library, 64<<20)
	request := map[string]any{"source_root": sources, "source": map[string]any{"id": "amd-69290-rev-1-00", "path": "amd-69290.pdf", "bytes": len(b), "digest": "sha256:" + hash(b)}, "decoder_root": decoder, "decoder": map[string]any{"path": a.Library, "digest": "sha256:" + hash(libBytes)}, "profile": l.Source.Profile}
	writeJSON(filepath.Join(root, "request.json"), request)
	writeJSON(filepath.Join(root, "assets.json"), map[string]any{"format_version": 1, "platform": platform, "required_profile_asset": a.Required, "pdfium_version": l.Version, "lock_sha256": hash(raw), "archive": a, "source_sha256": hash(b), "source_acquisition": acquisition, "runtime_execution": "not_yet_tested", "physical_hardware_qualification": false})
	fmt.Println("PASS exact decoder architecture, archive/runtime/license identities, source bytes; execution remains separate")
}
