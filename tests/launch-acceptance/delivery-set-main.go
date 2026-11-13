package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var o dsOptions
	flag.StringVar(&o.mode, "mode", "verify", "prepare, validate, verify, inspect-owner, inspect-native, or inspect-executable")
	flag.StringVar(&o.root, "root", "", "absolute private delivery-set directory")
	flag.StringVar(&o.selection, "selection", "", "independent exact artifact selection JSON outside delivery root")
	flag.StringVar(&o.policy, "allowed-signers", "", "independent allowed-signers policy outside delivery root")
	flag.StringVar(&o.profile, "profile", "", "explicit expected target profile")
	flag.StringVar(&o.signer, "signer", "", "expected signer principal for verification")
	flag.StringVar(&o.definition, "definition", "", "independent preparation definition outside delivery root")
	flag.StringVar(&o.output, "report", "", "optional new absolute result file, preferably outside delivery root")
	flag.StringVar(&o.artifact, "artifact", "", "explicit artifact ID for inspect-owner")
	flag.StringVar(&o.ownerTool, "owner-tool", "", "independent trusted read-only owner harness for inspect-owner")
	flag.StringVar(&o.ownerBuild, "owner-build", "", "independent exact owner harness build record for inspect-owner")
	flag.StringVar(&o.ownerRepository, "owner-repository", "", "independent reviewed native verifier source checkout")
	flag.StringVar(&o.ownerWork, "owner-work", "", "new private native extraction directory outside source and delivery")
	flag.StringVar(&o.nativePolicy, "native-allowed-signers", "", "independent native artifact signer policy")
	flag.StringVar(&o.nativeSigner, "native-signer", "", "independent native artifact signer principal")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	var r any
	var e error
	if o.mode == "inspect-owner" {
		r, e = dsInspectOwner(o)
	} else if o.mode == "inspect-native" {
		r, e = dsInspectNative(o)
	} else if o.mode == "inspect-executable" {
		r, e = dsInspectExecutable(o)
	} else {
		if o.artifact != "" || o.ownerTool != "" || o.ownerBuild != "" || o.ownerRepository != "" || o.ownerWork != "" || o.nativePolicy != "" || o.nativeSigner != "" {
			fmt.Fprintln(os.Stderr, dsError)
			os.Exit(1)
		}
		r, e = dsRun(o)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, dsError)
		os.Exit(1)
	}
	fmt.Print(string(dsJSON(r)))
}
