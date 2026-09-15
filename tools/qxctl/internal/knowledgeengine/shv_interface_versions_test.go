package knowledgeengine

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestSHVInterfaceOwnersInstalled(t *testing.T) {
	prefix := os.Getenv("SHV_INTERFACE_SET_PREFIX")
	if prefix == "" {
		t.Skip("select the five interface-owner packages")
	}
	ctx := context.Background()
	for name, invoke := range map[string]func() (Response, error){
		"kernel": func() (Response, error) {
			return InvokeSHV(ctx, prefix, SHVKernelInterfaceVersion, t.TempDir(), "inspect", []byte(`{}`))
		},
		"pdf": func() (Response, error) {
			return InvokeSHVPDF(ctx, prefix, SHVPDFInterfaceVersion, t.TempDir(), "inspect", []byte(`{}`))
		},
		"partition": func() (Response, error) {
			return InvokeSHVPartition(ctx, prefix, SHVPartitionInterfaceVersion, t.TempDir(), "inspect", []byte(`{}`))
		},
		"adapter": func() (Response, error) {
			return InvokeSHVGraphAdapter(ctx, prefix, SHVGraphAdapterInterfaceVersion, t.TempDir(), "inspect", []byte(`{}`))
		},
		"store": func() (Response, error) {
			return InvokeSHVStore(ctx, prefix, SHVStoreInterfaceVersion, t.TempDir(), "inspect", []byte(`{}`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := invoke(); err != nil {
				t.Fatal(err)
			}
		})
	}
	p, _ := shvTableFixture(t)
	root := shvText(p["source_root"])
	r, err := InvokeSHV(ctx, prefix, SHVKernelInterfaceVersion, root, "catalogue_build", shvRaw(t, p))
	if err != nil {
		t.Fatal(err)
	}
	cat, err := shvObject(r.Result)
	if err != nil {
		t.Fatal(err)
	}
	input := shvRaw(t, map[string]any{"source_root": root, "catalogue": cat})
	r, err = InvokeSHV(ctx, prefix, SHVKernelInterfaceVersion, root, "graph_project", input)
	if err != nil {
		t.Fatal(err)
	}
	if ValidateSHVResultVersion("graph_project", input, r.Result, false, SHVDocumentVersion) == nil {
		t.Fatal("old kernel accepted a new owner identity")
	}
}

func TestSHVTransferReaderVersionBoundaries(t *testing.T) {
	p := shvMap(transferFixture(t)["input"])
	if err := shvStoreInputVersion("transfer_plan", p, SHVStoreTransferVersion); err != nil {
		t.Fatal(err)
	}
	if shvStoreInputVersion("transfer_plan", p, SHVStoreInterfaceVersion) == nil {
		t.Fatal("new reader substituted for selected old reader")
	}
	writer := shvMap(p["target_connector"])
	old := shvText(writer["Version"])
	writer["Version"] = SHVStoreInterfaceVersion
	for _, key := range []string{"ReceiptPath", "ExecutablePath"} {
		writer[key] = strings.ReplaceAll(shvText(writer[key]), old, SHVStoreInterfaceVersion)
	}
	// The old selected reader must reject a new writer even before receipt replay.
	if shvStoreInputVersion("transfer_plan", p, SHVStoreTransferVersion) == nil {
		t.Fatal("old reader admitted a new writer")
	}
	source := shvMap(p["source_connector"])
	source["Version"] = SHVStoreInterfaceVersion
	for _, key := range []string{"ReceiptPath", "ExecutablePath"} {
		source[key] = strings.ReplaceAll(shvText(source[key]), SHVStoreTransferVersion, SHVStoreInterfaceVersion)
	}
	if err := shvStoreInputVersion("transfer_plan", p, SHVStoreInterfaceVersion); err != nil {
		t.Fatal(err)
	}
}
