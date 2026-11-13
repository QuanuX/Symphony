// Private local qualification of real Go owner packages; no provider/service work.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type qOwner struct {
	component, binary, adapter string
	leading                    []string
	record                     qObject
	executable                 string
}

func main() {
	qmain(func() {
		build := flag.String("build-root", "", "trusted local candidate builds")
		work := flag.String("work", "", "new absolute work")
		tool := flag.String("delivery-tool", "", "delivery verifier")
		flag.Parse()
		qneed(flag.NArg() == 0 && *build != "" && *work != "" && *tool != "", "required arguments missing")
		qualifyOwners(*build, *work, *tool)
	})
}
func qualifyOwners(buildRoot, destination, deliveryTool string) {
	buildRoot = qresolve(buildRoot)
	deliveryTool = qresolve(deliveryTool)
	work := qnewwork(destination, buildRoot)
	root := qjoin(work, "delivery")
	home := qjoin(work, "private-home")
	qerr(os.Mkdir(root, 0700))
	qerr(os.Mkdir(home, 0700))
	path := os.Getenv("PATH")
	if path == "" {
		path = "/usr/bin:/bin"
	}
	env := []string{"PATH=" + path, "HOME=" + home, "XDG_CONFIG_HOME=" + qjoin(home, "config"), "XDG_STATE_HOME=" + qjoin(home, "state"), "XDG_RUNTIME_DIR=" + qjoin(home, "runtime"), "LANG=C"}
	r := qnewRunner(work, 120*time.Second, env)
	owners := []qOwner{{component: "secure-identity-access-governance", binary: "symphony-ssiag", adapter: "ssiag-receipt-v2", leading: []string{"package"}}, {component: "stav-append-authority", binary: "symphony-stav-append-authority", adapter: "stav-receipt-v2"}, {component: "accordare-stav-producer", binary: "symphony-accordare-stav-producer", adapter: "accordare-receipt-v2"}}
	for i := range owners {
		o := &owners[i]
		o.record = qread(qjoin(buildRoot, o.component, "owner-tool-build.json"))
		identity := qread(qjoin(buildRoot, "identities", o.component+".json"))
		tool := qjoin(buildRoot, o.component, "bin/inspect-owner")
		o.executable = qjoin(buildRoot, o.component, "bin", o.binary)
		source := qo(o.record["source"])
		qneed(o.record["component_id"] == o.component && o.record["verification_adapter"] == o.adapter && o.record["tool_sha256"] == qdigest(tool) && identity["executable_sha256"] == qdigest(o.executable) && source["commit"] == identity["source_commit"] && source["tree"] == identity["source_tree"] && o.record["version"] == identity["version"] && o.record["source_archive_sha256"] == qdigest(qjoin(buildRoot, o.component, "source.tar")), "local build binding mismatch")
	}
	source := owners[0].record["source"]
	targetOS := qs(owners[0].record["os"])
	targetArch := qs(owners[0].record["architecture"])
	for _, o := range owners {
		qneed(qequal(o.record["source"], source) && o.record["os"] == targetOS && o.record["architecture"] == targetArch, "mixed build source or host")
	}
	profile := qObject{"profile_id": "local-owner-" + targetOS + "-" + targetArch, "os": targetOS, "architecture": targetArch}
	choices := []qObject{}
	artifacts := []qObject{}
	files := []string{}
	for _, row := range [][2]string{{"install.txt", "Fixture: existing explicit-prefix package install; no enrollment."}, {"remove.txt", "Fixture: owner uninstall on a disposable copy; preserve sentinel."}, {"notices.txt", "Local qualification only; actual redistribution notice review remains pending."}} {
		relative := "metadata/" + row[0]
		qwrite(qjoin(root, relative), []byte(row[1]))
		files = append(files, relative)
	}
	prefix := qjoin(root, "payload")
	qerr(os.Mkdir(prefix, 0700))
	for _, o := range owners {
		version := qs(o.record["version"])
		command := qappend([]string{o.executable}, o.leading...)
		r.run(o.component+"-install", qappend(command, "install", "--prefix", prefix, "--version", version), 0)
		primary := "payload/libexec/symphony/" + o.component + "/" + version + "/" + o.binary
		receipt := "payload/share/symphony/receipts/" + o.component + "/" + version + "/install-receipt.json"
		build := "metadata/" + o.component + "-build.json"
		material := "metadata/" + o.component + "-source.tar"
		qcopy(qjoin(buildRoot, "identities", o.component+".json"), qjoin(root, build), 0600, false)
		qcopy(qjoin(buildRoot, o.component, "source.tar"), qjoin(root, material), 0600, false)
		dependencies := []string{}
		if o.component == "accordare-stav-producer" {
			dependencies = []string{"secure-identity-access-governance", "stav-append-authority"}
		}
		choice := qObject{"id": o.component, "package_id": o.component, "kind": "go", "verification_adapter": o.adapter, "component_versions": qObject{o.component: version}, "dependencies": dependencies}
		choices = append(choices, choice)
		artifacts = append(artifacts, qObject{"selection": choice, "source": source, "vendor": nil, "file_paths": []string{primary, receipt}, "primary_path": primary, "payload_inventory_path": receipt, "manifest_path": nil, "signature_path": nil, "artifact_signer_policy_sha256": nil, "receipt_paths": []string{receipt}, "install_recipe_path": "metadata/install.txt", "remove_recipe_path": "metadata/remove.txt", "notice_paths": []string{"metadata/notices.txt"}, "source_material_paths": []string{material}, "build_evidence_paths": []string{build}})
		files = append(files, primary, receipt, build, material)
	}
	qwrite(qjoin(work, "selection.json"), qObject{"protocol": "symphony.release.delivery-selection.v1", "source": source, "profile": profile, "capability_manifest_sha256": qhash([]byte("local three-owner fixture only")), "dependency_map_sha256": qdigest(qjoin(buildRoot, "mvp-dependencies.json")), "identity_catalog_sha256": qdigest(qjoin(buildRoot, "mvp-release-identities.json")), "artifacts": choices})
	qwrite(qjoin(work, "definition.json"), qObject{"set_id": "local-go-owner-qualification", "file_paths": files, "artifacts": artifacts})
	reports := []qObject{}
	func() {
		key, cleanup := qfixtureKey(work, r)
		defer cleanup()
		r.run("fixture-key", []string{"ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key}, 0)
		qwrite(qjoin(work, "fixture-allowed-signers"), append([]byte("fixture namespaces=\"symphony-delivery-set\" "), qbytes(key+".pub")...))
		common := []string{deliveryTool, "-root", root, "-selection", qjoin(work, "selection.json"), "-allowed-signers", qjoin(work, "fixture-allowed-signers"), "-profile", qs(profile["profile_id"]), "-signer", "fixture"}
		r.run("prepare", qappend(common, "-mode", "prepare", "-definition", qjoin(work, "definition.json")), 0)
		r.run("fixture-sign", []string{"ssh-keygen", "-Y", "sign", "-f", key, "-n", "symphony-delivery-set", qjoin(root, "delivery-set.json")}, 0)
		for _, o := range owners {
			report := qjoin(work, o.component+"-owner.json")
			ownerBuild := qjoin(buildRoot, o.component, "owner-tool-build.json")
			r.run(o.component+"-owner", qappend(common, "-mode", "inspect-owner", "-artifact", o.component, "-owner-tool", qjoin(buildRoot, o.component, "bin/inspect-owner"), "-owner-build", ownerBuild, "-report", report), 0)
			qproof(qread(report))
			reports = append(reports, qObject{"component_id": o.component, "report_sha256": qdigest(report), "tool_build_sha256": qdigest(ownerBuild)})
		}
		receipt := qjoin(root, artifacts[0]["receipt_paths"].([]string)[0])
		original := qbytes(receipt)
		qreplace(receipt, []byte("tampered receipt"))
		r.run("tampered-receipt", qappend(common, "-mode", "verify"), 1)
		qreplace(receipt, original)
		r.run("restored-set", qappend(common, "-mode", "verify"), 0)
	}()
	disposable := qjoin(work, "removal-copy")
	qcopytree(prefix, disposable)
	sentinel := qjoin(disposable, "user-sentinel")
	qwrite(sentinel, []byte("preserve unrelated user data"))
	before := qdigest(sentinel)
	for i := len(owners) - 1; i >= 0; i-- {
		o := owners[i]
		command := qappend([]string{o.executable}, o.leading...)
		r.run(o.component+"-remove", qappend(command, "uninstall", "--prefix", disposable, "--version", qs(o.record["version"])), 0)
	}
	entries, e := os.ReadDir(home)
	qerr(e)
	qneed(qequal(qfiles(disposable), []string{"user-sentinel"}) && qdigest(sentinel) == before && len(entries) == 0, "removal or no-activation boundary failed")
	result := qObject{"protocol": "symphony.release.go-owner-qualification.v1", "status": "passed_local_source_qualification", "source": source, "profile": profile, "delivery_index_sha256": qdigest(qjoin(root, "delivery-set.json")), "delivery_tool_sha256": qdigest(deliveryTool), "owner_reports": reports, "commands": r.commands, "sentinel_preserved": true, "operational_state_created": false, "fixture_signing_key_destroyed": true, "skips": 0, "fresh_host_qualified": false, "provider_qualified": false, "notice_review_complete": false, "release_acceptance_complete": false, "publication_authorized": false}
	qwrite(qjoin(work, "evidence.json"), result)
	b, e := json.Marshal(qObject{"status": result["status"], "owner_reports": len(reports), "release_acceptance_complete": false, "publication_authorized": false})
	qerr(e)
	fmt.Println(string(b))
}
