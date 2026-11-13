// Exercise the native delivery adapter with the existing package-release fixture.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"time"
)

var qNativeSources = []string{"tests/launch-acceptance/candidate-host.go", "cmake/release/VerifyCandidate.cmake", "cmake/release/Support.cmake", "cmake/release/StrictJSON.cmake", "cmake/SymphonyReceiptV2Support.cmake"}

func main() {
	qmain(func() {
		paths := map[string]*string{}
		for _, name := range []string{"repository", "candidate", "candidate-host", "delivery-tool", "policy", "key", "work"} {
			paths[name] = flag.String(name, "", "explicit local "+name)
		}
		flag.Parse()
		qneed(flag.NArg() == 0, "unexpected arguments")
		resolved := map[string]string{}
		for name, path := range paths {
			qneed(*path != "", "required argument missing: "+name)
			if name != "work" {
				resolved[name] = qresolve(*path)
			}
		}
		qualifyNative(resolved, *paths["work"])
	})
}
func qualifyNative(paths map[string]string, destination string) {
	work := qnewwork(destination, paths["repository"], paths["candidate"])
	root := qjoin(work, "delivery")
	qerr(os.Mkdir(root, 0700))
	qcopytree(paths["candidate"], qjoin(root, "native"))
	manifest := qread(qjoin(root, "native/candidate.json"))
	versions := qObject{}
	components := qa(manifest["components"])
	for _, row := range components {
		c := qo(row)
		id := qs(c["component_id"])
		qneed(versions[id] == nil, "select one version per component")
		versions[id] = qs(c["version"])
	}
	platform := qo(manifest["platform"])
	profile := qObject{"profile_id": platform["profile_id"], "os": platform["os"], "architecture": platform["architecture"]}
	choice := qObject{"id": "native-fixture", "package_id": "native-fixture", "kind": "native", "verification_adapter": "native-candidate-v1", "component_versions": versions, "dependencies": []string{}}
	entries, e := os.ReadDir(paths["candidate"])
	qerr(e)
	files := []string{}
	for _, entry := range entries {
		files = append(files, "native/"+entry.Name())
	}
	sort.Strings(files)
	metadata := "metadata/fixture-provenance.json"
	qwrite(qjoin(root, metadata), qObject{"scope": "synthetic native package fixture", "source": manifest["source"], "notice_review_complete": false})
	artifact := qObject{"selection": choice, "source": manifest["source"], "vendor": nil, "file_paths": files, "primary_path": "native/candidate.tar", "payload_inventory_path": "native/candidate.json", "manifest_path": "native/candidate.json", "signature_path": "native/candidate.json.sig", "artifact_signer_policy_sha256": qdigest(paths["policy"]), "receipt_paths": []string{}, "install_recipe_path": metadata, "remove_recipe_path": metadata, "notice_paths": []string{metadata}, "source_material_paths": []string{metadata}, "build_evidence_paths": []string{"native/build-evidence.json"}}
	qwrite(qjoin(work, "selection.json"), qObject{"protocol": "symphony.release.delivery-selection.v1", "source": manifest["source"], "profile": profile, "capability_manifest_sha256": qdigest(qjoin(root, metadata)), "dependency_map_sha256": manifest["dependency_map_sha256"], "identity_catalog_sha256": manifest["identity_catalog_sha256"], "artifacts": []qObject{choice}})
	qwrite(qjoin(work, "definition.json"), qObject{"set_id": "native-fixture", "file_paths": append(append([]string{}, files...), metadata), "artifacts": []qObject{artifact}})
	hashes := qObject{}
	for _, path := range qNativeSources {
		hashes[path] = qdigest(qjoin(paths["repository"], path))
	}
	qwrite(qjoin(work, "native-tool.json"), qObject{"protocol": "symphony.release.native-verifier-tool.v1", "tool_sha256": qdigest(paths["candidate-host"]), "go_version": "go1.26.5", "source_files_sha256": hashes})
	r := qnewRunner(work, 720*time.Second, nil)
	common := []string{paths["delivery-tool"], "-root", root, "-selection", qjoin(work, "selection.json"), "-allowed-signers", paths["policy"], "-profile", qs(profile["profile_id"]), "-signer", "fixture-publisher"}
	r.run("prepare", qappend(common, "-mode", "prepare", "-definition", qjoin(work, "definition.json")), 0)
	r.run("sign-index", []string{"ssh-keygen", "-Y", "sign", "-f", paths["key"], "-n", "symphony-delivery-set", qjoin(root, "delivery-set.json")}, 0)
	inspect := qappend(common, "-mode", "inspect-native", "-artifact", "native-fixture", "-owner-tool", paths["candidate-host"], "-owner-build", qjoin(work, "native-tool.json"), "-owner-repository", paths["repository"], "-native-allowed-signers", paths["policy"], "-native-signer", "fixture-publisher")
	r.run("inspect", qappend(inspect, "-owner-work", qjoin(work, "extracted"), "-report", qjoin(work, "native-owner.json")), 0)
	proof := qread(qjoin(work, "native-owner.json"))
	qproof(proof)
	qneed(proof["artifact_sha256"] == qo(manifest["artifact"])["sha256"], "native artifact mismatch")
	r.run("reused-work", qappend(inspect, "-owner-work", qjoin(work, "extracted")), 1)
	policy := qjoin(work, "changed-native-policy")
	qwrite(policy, append(qbytes(paths["policy"]), '\n'))
	r.run("wrong-native-policy", qappend(inspect, "-native-allowed-signers", policy, "-owner-work", qjoin(work, "must-not-extract")), 1)
	qneed(!qexists(qjoin(work, "must-not-extract")), "native policy mismatch performed extraction")
	qwrite(qjoin(work, "evidence.json"), qObject{"protocol": "symphony.release.native-delivery-qualification.v1", "status": "passed_fixture_only", "source": manifest["source"], "profile": profile, "native_owner_report_sha256": qdigest(qjoin(work, "native-owner.json")), "tool_record_sha256": qdigest(qjoin(work, "native-tool.json")), "cases": []string{"signed_native_receipts", "reused_work_refusal", "independent_native_policy_refusal"}, "fresh_host_qualified": false, "release_acceptance_complete": false, "publication_authorized": false})
	fmt.Println("Native delivery adapter passed fixture extraction and refusal checks; acceptance remains pending")
}
