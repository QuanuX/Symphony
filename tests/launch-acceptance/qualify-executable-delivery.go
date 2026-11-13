// Qualify a trusted local qxctl or plain SOV executable in a fixture-signed set.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	qmain(func() {
		binary := flag.String("binary", "", "trusted local executable")
		identity := flag.String("identity", "", "local build identity")
		delivery := flag.String("delivery-tool", "", "delivery verifier")
		destination := flag.String("work", "", "new absolute work")
		flag.Parse()
		qneed(flag.NArg() == 0 && *binary != "" && *identity != "" && *delivery != "" && *destination != "", "required arguments missing")
		qualifyExecutable(*binary, *identity, *delivery, *destination)
	})
}
func qualifyExecutable(binary, identity, delivery, destination string) {
	build := qread(identity)
	component := qs(build["component_id"])
	qneed(component == "qxctl" || component == "sov-stdio-adapter" || component == "sov-ssh-adapter", "unsupported executable")
	qneed(qdigest(binary) == qs(build["executable_sha256"]), "binary does not match selected local build")
	work := qnewwork(destination)
	root := qjoin(work, "delivery")
	qerr(os.Mkdir(root, 0700))
	r := qnewRunner(work, 90*time.Second, nil)
	source := qObject{"commit": build["source_commit"], "tree": build["source_tree"]}
	osName := qs(build["goos"])
	if osName == "darwin" {
		osName = "macos"
	}
	profile := qObject{"profile_id": "local-executable-" + osName + "-" + qs(build["goarch"]), "os": osName, "architecture": build["goarch"]}
	primary := "payload/" + component
	qcopy(binary, qjoin(root, primary), 0700, false)
	info, e := os.Stat(qjoin(root, primary))
	qerr(e)
	inventory := "payload/immutable-files.json"
	qwrite(qjoin(root, inventory), qObject{"protocol": "symphony.release.immutable-files.v1", "files": []qObject{{"path": primary, "sha256": qdigest(qjoin(root, primary)), "size": info.Size(), "mode": 0700}}})
	identityPath := "metadata/build-identity.json"
	qwrite(qjoin(root, identityPath), qbytes(identity))
	metadata := "metadata/fixture.json"
	qwrite(qjoin(root, metadata), qObject{"scope": "local executable fixture", "source": source, "notice_review_complete": false, "install_contract": "explicit selected executable path"})
	adapter := "sov-executable-v1"
	if component == "qxctl" {
		adapter = "qxctl-files-v1"
	}
	choice := qObject{"id": component, "package_id": component, "kind": "go", "verification_adapter": adapter, "component_versions": qObject{component: build["version"]}, "dependencies": []string{}}
	artifact := qObject{"selection": choice, "source": source, "vendor": nil, "file_paths": []string{primary, inventory}, "primary_path": primary, "payload_inventory_path": inventory, "manifest_path": nil, "signature_path": nil, "artifact_signer_policy_sha256": nil, "receipt_paths": []string{}, "install_recipe_path": metadata, "remove_recipe_path": metadata, "notice_paths": []string{metadata}, "source_material_paths": []string{metadata}, "build_evidence_paths": []string{identityPath}}
	qwrite(qjoin(work, "selection.json"), qObject{"protocol": "symphony.release.delivery-selection.v1", "source": source, "profile": profile, "capability_manifest_sha256": qdigest(qjoin(root, metadata)), "dependency_map_sha256": build["dependency_map_sha256"], "identity_catalog_sha256": build["release_identity_catalog_sha256"], "artifacts": []qObject{choice}})
	qwrite(qjoin(work, "definition.json"), qObject{"set_id": "local-executable-fixture", "file_paths": []string{primary, inventory, metadata, identityPath}, "artifacts": []qObject{artifact}})
	func() {
		key, cleanup := qfixtureKey(work, r)
		defer cleanup()
		r.run("fixture-key", []string{"ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key}, 0)
		qwrite(qjoin(work, "allowed-signers"), append([]byte("fixture namespaces=\"symphony-delivery-set\" "), qbytes(key+".pub")...))
		common := []string{qresolve(delivery), "-root", root, "-selection", qjoin(work, "selection.json"), "-allowed-signers", qjoin(work, "allowed-signers"), "-profile", qs(profile["profile_id"]), "-signer", "fixture"}
		r.run("prepare", qappend(common, "-mode", "prepare", "-definition", qjoin(work, "definition.json")), 0)
		r.run("sign", []string{"ssh-keygen", "-Y", "sign", "-f", key, "-n", "symphony-delivery-set", qjoin(root, "delivery-set.json")}, 0)
		inspect := qappend(common, "-mode", "inspect-executable", "-artifact", component)
		report := qjoin(work, "executable-owner.json")
		r.run("inspect", qappend(inspect, "-report", report), 0)
		qproof(qread(report))
		r.run("reused-output", qappend(inspect, "-report", report), 1)
		f, e := os.OpenFile(qjoin(root, primary), os.O_WRONLY|os.O_APPEND, 0700)
		qerr(e)
		_, e = f.Write([]byte("tampered"))
		closeErr := f.Close()
		qerr(e)
		qerr(closeErr)
		r.run("tampered-executable", inspect, 1)
		qcopy(binary, qjoin(root, primary), 0700, true)
		r.run("restored-set", qappend(common, "-mode", "verify"), 0)
	}()
	qwrite(qjoin(work, "evidence.json"), qObject{"protocol": "symphony.release.executable-qualification.v1", "status": "passed_local_source_qualification", "source": source, "profile": profile, "component_id": component, "executable_sha256": build["executable_sha256"], "owner_report_sha256": qdigest(qjoin(work, "executable-owner.json")), "cases": []string{"signed_identity_and_version", "reused_output_refusal", "tamper_refusal", "restored_set"}, "fixture_key_destroyed": true, "fresh_host_qualified": false, "protected_profile_qualified": false, "release_acceptance_complete": false, "publication_authorized": false})
	fmt.Println(component + ": real executable identity and tamper checks passed; final-host acceptance pending")
}
