package knowledgeengine

import (
	"context"
	"os"
	"testing"
)

func TestSHVSourceInterfaceInstalledReplay(t *testing.T) {
	prefix := os.Getenv("SHV_OWNER_INTERFACE_PREFIX")
	if prefix == "" {
		t.Skip("set SHV_OWNER_INTERFACE_PREFIX to the source/publication interface installation")
	}
	plan, _, capture, _ := lifecycleFixture(t)
	invoke := func(op string, p any) map[string]any {
		t.Helper()
		r, e := InvokeSHVSource(context.Background(), prefix, SHVSourceInterfaceVersion, shvText(capture["source_root"]), op, shvRaw(t, p))
		if e != nil {
			t.Fatal(op, e)
		}
		v, e := shvObject(r.Result)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	invoke("inspect", map[string]any{})
	invoke("source_plan", plan)
	c := invoke("capture_import", capture)
	input := map[string]any{"source_root": capture["source_root"], "captures": []any{c}}
	graph := invoke("graph_project", input)
	invoke("graph_validate", map[string]any{"source_root": capture["source_root"], "graph": graph})
	if ValidateSHVSourceResult("graph_project", shvRaw(t, input), shvRaw(t, graph)) == nil {
		t.Fatal("old source owner admitted new graph identity")
	}
	bad := shvClone(t, graph)
	shvMap(bad["owner"])["engine_version"] = SHVSourceVersion
	bad = shvReseal(t, bad)
	if ValidateSHVSourceResultVersion("graph_project", shvRaw(t, input), shvRaw(t, bad), SHVSourceInterfaceVersion) == nil {
		t.Fatal("resealed owner substitution accepted")
	}
	if _, e := InvokeSHVSource(context.Background(), prefix, "latest", "", "inspect", []byte(`{}`)); e == nil {
		t.Fatal("unknown owner release accepted")
	}
}

func TestSHVPublicationInterfaceInstalledReplay(t *testing.T) {
	prefix := os.Getenv("SHV_OWNER_INTERFACE_PREFIX")
	if prefix == "" {
		t.Skip("set SHV_OWNER_INTERFACE_PREFIX to the source/publication interface installation")
	}
	input, _ := publicationFixture(t)
	for _, op := range []string{"inspect", "publication_plan"} {
		p := input
		if op == "inspect" {
			p = map[string]any{}
		}
		r, e := InvokeSHVPublication(context.Background(), prefix, SHVPublicationInterfaceVersion, t.TempDir(), op, shvRaw(t, p))
		if e != nil {
			t.Fatal(e)
		}
		if op == "publication_plan" {
			v, e := shvObject(r.Result)
			if e != nil {
				t.Fatal(e)
			}
			v["reason"] = "forged"
			v = shvReseal(t, v)
			if ValidateSHVPublicationResultVersion(op, shvRaw(t, p), shvRaw(t, v), SHVPublicationInterfaceVersion) == nil {
				t.Fatal("resealed false plan accepted")
			}
		}
	}
}
