//go:build ignore

// Required-gate evidence: reject an omitted/skipped test and bind actual owners.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func bytes(p string) []byte { b, e := os.ReadFile(p); must(e); return b }
func identity(p string) map[string]any {
	b := bytes(p)
	return map[string]any{"path": p, "bytes": len(b), "sha256": fmt.Sprintf("%x", sha256.Sum256(b))}
}
func main() {
	work := flag.String("work", "", "campaign work")
	prefix := flag.String("prefix", "", "owners")
	store := flag.String("store-prefix", "", "store")
	request := flag.String("request", "", "request")
	repository := flag.String("repository", "", "source")
	flag.Parse()
	f, e := os.Open(filepath.Join(*work, "logs/catalogue.jsonl"))
	must(e)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 65536), 4<<20)
	runs, passes := 0, 0
	for scanner.Scan() {
		var v struct{ Action, Test string }
		must(json.Unmarshal(scanner.Bytes(), &v))
		if v.Action == "skip" || v.Action == "fail" {
			panic("required catalogue test skipped or failed")
		}
		if v.Test == "TestInstalledSHVBoundedCatalogueWorkflow" {
			if v.Action == "run" {
				runs++
			}
			if v.Action == "pass" {
				passes++
			}
		}
	}
	must(scanner.Err())
	if runs != 1 || passes != 1 {
		panic("required catalogue test did not execute exactly once")
	}
	var summary map[string]any
	must(json.Unmarshal(bytes(filepath.Join(*work, "catalogue/summary.json")), &summary))
	if summary["status"] != "passed" || summary["skips"] != float64(0) {
		panic("missing catalogue evidence")
	}
	var receipts []any
	for _, owner := range []struct{ id, version string }{{"shv-engine", "0.4.0-dev"}, {"shv-pdf-adapter", "0.3.0-dev"}, {"shv-source-engine", "0.2.0-dev"}, {"shv-profile-engine", "0.3.0-dev"}, {"shv-partition-engine", "0.4.0-dev"}, {"shv-publication-engine", "0.4.0-dev"}, {"shv-graph-duckdb-connector", "0.4.0-dev"}} {
		base := *prefix
		if owner.id == "shv-graph-duckdb-connector" {
			base = *store
		}
		receipts = append(receipts, identity(filepath.Join(base, "share/symphony/receipts", owner.id, owner.version, "install-receipt.json")))
	}
	result := map[string]any{"status": "passed", "goos": runtime.GOOS, "goarch": runtime.GOARCH, "go_version": runtime.Version(), "source": string(bytes(filepath.Join(*work, "source.txt"))), "source_status": string(bytes(filepath.Join(*work, "source-status.txt"))), "lock": identity(filepath.Join(*repository, "tests/shv-release-qualification/dependencies.json")), "request": identity(*request), "owner_receipts": receipts, "qxctl": identity(filepath.Join(*work, "bin/qxctl")), "ssiag": identity(filepath.Join(*work, "bin/ssiag")), "stav": identity(filepath.Join(*work, "bin/stav")), "catalogue": summary, "installed_pdf_replay": identity(filepath.Join(*work, "pdf-replay/ACCEPTANCE.json")), "physical_hardware_tested": false, "publication_applied": false}
	b, e := json.MarshalIndent(result, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(*work, "summary.json"), append(b, '\n'), 0600))
	fmt.Println("PASS: required SHV installed PDF and protected catalogue workflow; zero skips; publication preview only")
}
