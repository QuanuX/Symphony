// Dependency evidence collector. It reads artifact metadata; it never executes
// a selected native binary, invokes ldd, grants a license, or publishes a package.
package main

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type object = map[string]any

var repo, prefix, output, cache, goroot string
var notices []object

func fail(e error) {
	if e != nil {
		panic(e)
	}
}
func read(p string) []byte   { b, e := os.ReadFile(p); fail(e); return b }
func hash(b []byte) string   { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func decode(p string) object { var j object; fail(json.Unmarshal(read(p), &j)); return j }
func encode(j any) []byte    { b, e := json.MarshalIndent(j, "", "  "); fail(e); return append(b, '\n') }
func fileDigest(path string) string {
	f, e := os.Open(path)
	fail(e)
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	fail(e)
	return fmt.Sprintf("%x", h.Sum(nil))
}
func checked(root, rel string) string {
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "\\") {
		panic("invalid inventory relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		panic("inventory path escapes root")
	}
	p := filepath.Join(root, clean)
	st, e := os.Lstat(p)
	fail(e)
	if !st.Mode().IsRegular() {
		panic("inventory source is not a regular file: " + rel)
	}
	real, e := filepath.EvalSymlinks(p)
	fail(e)
	base, e := filepath.EvalSymlinks(root)
	fail(e)
	r, e := filepath.Rel(base, real)
	fail(e)
	if r == ".." || strings.HasPrefix(r, ".."+string(os.PathSeparator)) {
		panic("inventory source escapes selected root")
	}
	return p
}
func retain(source, rel, component, scope string) object {
	b := read(source)
	dest := filepath.Join(output, "notices", filepath.FromSlash(rel))
	fail(os.MkdirAll(filepath.Dir(dest), 0700))
	fail(os.WriteFile(dest, b, 0600))
	n := object{"component": component, "path": "notices/" + filepath.ToSlash(rel), "sha256": hash(b), "bytes": len(b), "scope": scope}
	notices = append(notices, n)
	return n
}
func isNotice(name string) bool {
	s := strings.ToUpper(name)
	for _, p := range []string{"LICENSE", "LICENCE", "COPYING", "COPYRIGHT", "NOTICE", "PATENTS", "UNLICENSE"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
func legalFiles(root string) []string {
	var names []string
	fail(filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() && isNotice(d.Name()) {
			rel, e := filepath.Rel(root, p)
			if e != nil {
				return e
			}
			names = append(names, filepath.ToSlash(rel))
		}
		return nil
	}))
	sort.Strings(names)
	return names
}
func loader(p string) object {
	f, e := os.Open(p)
	fail(e)
	b := make([]byte, 8)
	n, e := f.Read(b)
	fail(f.Close())
	if e != nil && e != io.EOF {
		panic(e)
	}
	b = b[:n]
	kind := "unknown"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if len(b) >= 8 && string(b[:8]) == "!<arch>\n" {
		return object{"kind": "static_archive", "linkage_scope": "archive members; final consumer linkage must be observed separately"}
	}
	if len(b) >= 4 && string(b[:4]) == "\x7fELF" {
		kind = "ELF"
		cmd = exec.CommandContext(ctx, "readelf", "-d", p)
	} else if runtime.GOOS == "darwin" {
		kind = "Mach-O"
		cmd = exec.CommandContext(ctx, "/usr/bin/otool", "-L", p)
	} else {
		return object{"kind": kind, "status": "not_a_supported_native_object"}
	}
	raw, e := cmd.CombinedOutput()
	if e != nil {
		panic(fmt.Sprintf("static loader inspection failed: %s: %v: %s", p, e, raw))
	}
	return object{"kind": kind, "static_loader_output": string(raw), "scope": "declared loader references, not resolved recursive runtime closure or TLS handshake"}
}
func receiptInventory() []object {
	var rows []object
	pinned := map[string]string{}
	for _, entry := range decode(filepath.Join(repo, "cmake/notices/sources.json"))["components"].([]any) {
		row := entry.(map[string]any)
		name := "curl-COPYING"
		if row["id"] == "zstd" {
			name = "zstd-LICENSE"
		}
		pinned[name] = row["notice_sha256"].(string)
	}
	duck := decode(filepath.Join(repo, "modules/scv-graph-duckdb-connector/DUCKDB-PROVENANCE.json"))["files"].(map[string]any)
	pinned["DuckDB-LICENSE.MIT"] = duck["LICENSE"].(map[string]any)["sha256"].(string)
	pinned["DuckDB-THIRD-PARTY-LICENSES.txt"] = duck["THIRD_PARTY_LICENSES.txt"].(map[string]any)["sha256"].(string)
	root := filepath.Join(prefix, "share", "symphony", "receipts")
	fail(filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() || d.Name() != "install-receipt.json" {
			return nil
		}
		rel, e := filepath.Rel(prefix, p)
		if e != nil {
			return e
		}
		checked(prefix, filepath.ToSlash(rel))
		j := decode(p)
		if j["protocol"] != "symphony.knowledge.install-receipt.v2" {
			return fmt.Errorf("unsupported receipt protocol: %s", rel)
		}
		id, ok := j["component_id"].(string)
		if !ok {
			return fmt.Errorf("missing component identity")
		}
		version, ok := j["version"].(string)
		if !ok {
			return fmt.Errorf("missing component version")
		}
		var artifacts, ownedNotices []object
		files, ok := j["files"].([]any)
		if !ok {
			return fmt.Errorf("missing receipt files")
		}
		names := map[string]bool{}
		for _, x := range files {
			f := x.(map[string]any)
			name := f["path"].(string)
			notice := strings.HasPrefix(name, "share/licenses/")
			native := f["kind"] == "executable" || strings.HasSuffix(name, ".a") || strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".dylib")
			if !notice && !native {
				continue
			}
			source := checked(prefix, name)
			st, e := os.Stat(source)
			if e != nil {
				return e
			}
			sha := fileDigest(source)
			if f["digest"] != "sha256:"+sha || f["size"] != float64(st.Size()) {
				return fmt.Errorf("receipt file differs: %s", name)
			}
			if notice {
				base := filepath.Base(name)
				if expected, ok := pinned[base]; ok && expected != sha {
					return fmt.Errorf("selected upstream notice differs: %s", name)
				}
				names[base] = true
				ownedNotices = append(ownedNotices, retain(source, "installed/"+name, id+"@"+version, "receipt-owned installed legal material"))
			}
			if native {
				artifacts = append(artifacts, object{"path": name, "sha256": sha, "bytes": st.Size(), "loader": loader(source)})
			}
		}
		required := map[string][]string{"native-source-support-cpp": {"curl-COPYING"}, "sqav-databento-dbn-cpp": {"curl-COPYING"}, "sqav-databento-reference-cpp": {"zstd-LICENSE"}, "sqav-request-engine": {"zstd-LICENSE", "curl-COPYING"}, "scv-graph-duckdb-connector": {"DuckDB-LICENSE.MIT", "DuckDB-THIRD-PARTY-LICENSES.txt"}, "shv-graph-duckdb-connector": {"DuckDB-LICENSE.MIT", "DuckDB-THIRD-PARTY-LICENSES.txt"}}
		for _, n := range required[id] {
			if !names[n] {
				return fmt.Errorf("required notice absent from %s receipt: %s", id, n)
			}
		}
		rows = append(rows, object{"component_id": id, "version": version, "receipt_path": filepath.ToSlash(rel), "receipt_file_sha256": hash(read(p)), "declared_receipt_digest": j["receipt_digest"], "verified_scope": "exact owned notice and artifact bytes; full canonical receipt validation is the preceding installed gate", "artifacts": artifacts, "notices": ownedNotices})
		return nil
	}))
	sort.Slice(rows, func(i, j int) bool { return rows[i]["component_id"].(string) < rows[j]["component_id"].(string) })
	return rows
}
func escapeModule(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(c + 'a' - 'A')
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}
func moduleMaterial(path, version string) []object {
	if strings.Contains(path, "..") || strings.Contains(version, "/") || filepath.IsAbs(path) {
		panic("invalid Go module identity")
	}
	root := filepath.Join(cache, escapeModule(path)+"@"+escapeModule(version))
	if _, e := os.Stat(root); os.IsNotExist(e) {
		return nil
	}
	var found []object
	for _, rel := range legalFiles(root) {
		found = append(found, retain(checked(root, rel), "go/"+hash([]byte(path+"@"+version))+"/"+rel, path+"@"+version, "legal files retained from exact verified module cache; review their applicability to linked packages"))
	}
	return found
}
func goArtifact(p string) object {
	info, e := buildinfo.ReadFile(p)
	fail(e)
	rows := []object{}
	for _, m := range info.Deps {
		if m.Replace != nil {
			panic("selected Go artifact contains replacement module")
		}
		material := moduleMaterial(m.Path, m.Version)
		if len(material) == 0 {
			panic("linked Go module legal material unavailable: " + m.Path + "@" + m.Version)
		}
		rows = append(rows, object{"path": m.Path, "version": m.Version, "sum": m.Sum, "scope": "linked module reported by actual executable buildinfo; not every package in the module", "notices": material})
	}
	if strings.SplitN(strings.TrimSpace(string(read(filepath.Join(goroot, "VERSION")))), "\n", 2)[0] != info.GoVersion {
		panic("selected Go toolchain legal material does not match artifact GoVersion")
	}
	var standard []object
	for _, name := range []string{"LICENSE", "PATENTS"} {
		p := filepath.Join(goroot, name)
		if _, e := os.Stat(p); e == nil {
			standard = append(standard, retain(checked(goroot, name), "go-toolchain/"+name, info.GoVersion, "Go runtime/toolchain legal material; vendored subcomponents also retained"))
		}
	}
	vendor := filepath.Join(goroot, "src", "vendor")
	if _, e := os.Stat(vendor); e == nil {
		for _, name := range legalFiles(vendor) {
			standard = append(standard, retain(checked(vendor, name), "go-toolchain/vendor/"+name, info.GoVersion, "available standard-library vendor legal material; superset, not proof all files are linked"))
		}
	}
	mainNotice := retain(checked(repo, "LICENSE"), "symphony/LICENSE", info.Main.Path, "current main module source-root license; match final source commit before distribution")
	return object{"artifact": p, "sha256": fileDigest(p), "go_version": info.GoVersion, "main": info.Main, "settings": info.Settings, "linked_modules": rows, "runtime_notices": standard, "main_notice": mainNotice, "legal_approval": false}
}
func external(p string) object {
	if p == "" {
		return object{"status": "not_supplied"}
	}
	if _, e := os.Stat(p); os.IsNotExist(e) {
		return object{"status": "not_present_for_this_group"}
	}
	record := "share/symphony/dependency-evidence/curl-zstd.json"
	selected := checked(p, record)
	j := decode(selected)
	retain(selected, "external/curl-zstd-build-selection.json", "curl+zstd", "verified qualification recipe source pins and build-time selection; not runtime TLS proof")
	for _, pair := range []struct{ id, version string }{{"curl", "8.7.1"}, {"zstd", "1.5.7"}} {
		n := "share/licenses/" + pair.id + "/" + pair.version + "/LICENSE"
		retain(checked(p, n), "external/"+n, pair.id+"@"+pair.version, "selected exact upstream notice")
	}
	var objects []object
	fail(filepath.WalkDir(filepath.Join(p, "lib"), func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		name := d.Name()
		if (strings.Contains(name, "libcurl") || strings.Contains(name, "libzstd")) && (strings.HasSuffix(name, ".a") || strings.HasSuffix(name, ".dylib") || strings.Contains(name, ".so")) {
			objects = append(objects, object{"path": path, "sha256": fileDigest(path), "loader": loader(path)})
		}
		return nil
	}))
	return object{"status": "observed", "build_selection": j, "artifacts": objects, "unresolved": "Exact resolved TLS/transitive runtime files and matching license/source obligations remain publisher review; static loader references alone are insufficient."}
}
func main() {
	if len(os.Args) != 7 {
		fmt.Fprintln(os.Stderr, "usage: inventory REPO PREFIX OUTPUT EXTERNAL GO_ARTIFACT GOMODCACHE")
		os.Exit(2)
	}
	repo, prefix, output, cache = os.Args[1], os.Args[2], os.Args[3], os.Args[6]
	goroot = runtime.GOROOT()
	failed := false
	result := object{"format_version": 1, "publication_authorized": false, "release_ready": false, "publisher_review": "pending", "status": "in_progress"}
	defer func() {
		if r := recover(); r != nil {
			result["status"] = "failed"
			result["error"] = fmt.Sprint(r)
			fmt.Fprintln(os.Stderr, r)
			failed = true
		}
		result["notices"] = notices
		fail(os.WriteFile(filepath.Join(output, "inventory.json"), encode(result), 0600))
		if failed {
			os.Exit(1)
		}
	}()
	source := decode(filepath.Join(repo, "cmake/mvp-dependencies.json"))
	result["dependency_map_sha256"] = hash(read(filepath.Join(repo, "cmake/mvp-dependencies.json")))
	result["source_notice_manifest_sha256"] = fileDigest(filepath.Join(repo, "cmake/notices/sources.json"))
	result["source_scope"] = "catalog and exact installed artifact observations; final artifact source/build identity must be bound by the release record, not inferred from the current checkout"
	result["declared_external_dependencies"] = source["external_dependencies"]
	result["resolved_go_graphs_not_linked_claim"] = source["go_modules"]
	result["go_artifact_coverage"] = "Only the explicitly supplied executable is observed; source graphs list all planned commands. Collect each final distributed Go executable before release."
	result["installed_components"] = receiptInventory()
	result["external_build"] = external(os.Args[4])
	if p := os.Args[5]; p != "" {
		if _, e := os.Stat(p); e == nil {
			result["go_artifact"] = goArtifact(p)
		} else if !os.IsNotExist(e) {
			panic(e)
		} else {
			result["go_artifact"] = object{"status": "not_present_for_this_group"}
		}
	}
	result["optional_pdfium_selection"] = decode(filepath.Join(repo, "tests/shv-release-qualification/dependencies.json"))
	result["optional_pdfium_scope"] = "Exact already selected qualification archive/recipe and every supplied license hash; decoder is caller-selected and is not silently included in native receipts. Asset-stage verification is separate."
	result["remaining_release_gates"] = []string{"Publisher decides and signs off exact artifact/linkage/redistribution scope; no approval is inferred", "Supply matching Symphony corresponding source and build/install material for the exact distributed source and modifications", "Resolve actual TLS and every recursive runtime library to exact package/file identities and applicable notices/source terms", "For chosen PDFium binary retain all platform-specific license files and source/build provenance; review NOTICE/source obligations of its complete selected closure", "Retain Go toolchain and actual linked modules' notices; distinguish test/tool/resolved-only modules", "Keep proprietary IBKR SDK, paid/private market data and user plugins excluded unless separately authorized; public fixtures keep their own attribution"}
	result["status"] = "inventory_complete_publisher_review_pending"
	fmt.Printf("Notice inventory: %d installed components; %d retained legal/evidence files; publisher review pending\n", len(result["installed_components"].([]object)), len(notices))
}
