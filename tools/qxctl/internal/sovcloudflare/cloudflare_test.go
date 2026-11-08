package sovcloudflare

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloudflareStrictConsumerAndDigest(t *testing.T) {
	for _, raw := range []string{`{"a":1,"a":2}`, `{} {}`, `{"x":1.5}`, `{"x":1e2}`, `{"x":9007199254740992}`, `{"x":-0}`, `{"x":"\ud800"}`, `{"x":"\udc00"}`, string([]byte{0xff})} {
		if _, err := Decode([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid control %q", raw)
		}
	}
	fixture, err := os.ReadFile("testdata/digests.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		Value  json.RawMessage `json:"value"`
		Domain string          `json:"domain"`
		Digest string          `json:"digest"`
	}
	if err = json.Unmarshal(fixture, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors {
		value, err := Decode(v.Value)
		if err != nil {
			t.Fatal(err)
		}
		if got := Digest(value, v.Domain); got != v.Digest {
			t.Fatalf("independent digest = %s, want %s", got, v.Digest)
		}
	}
}

func TestCloudflareLostMutationNeverMeansNoEffect(t *testing.T) {
	r := map[string]any{"operation": "installation.apply"}
	out, err := Decode(lost(r))
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if m["outcome"] != "unknown" || m["effect_state"] != "unresolved" {
		t.Fatalf("lost mutation = %#v", m)
	}
	r["operation"] = "capability.list"
	out, _ = Decode(lost(r))
	if out.(map[string]any)["outcome"] != "failed" {
		t.Fatal("lost read became success")
	}
}

func TestCloudflareClosedResultSchema(t *testing.T) {
	schema, err := Decode([]byte(`{"type":"object","additionalProperties":false,"required":["outcome","data"],"properties":{"outcome":{"enum":["completed","rejected"]},"data":{"oneOf":[{"type":"null"},{"type":"object","additionalProperties":false,"required":["provider_dispatch"],"properties":{"provider_dispatch":{"const":false}}}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{"outcome":"completed","data":{"provider_dispatch":true}}`, `{"outcome":"SENTINEL","data":null}`, `{"outcome":"completed","data":null,"stderr":"SENTINEL"}`, `{"outcome":"completed","data":{"provider_dispatch":false,"secret":"SENTINEL"}}`} {
		value, err := Decode([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if matches(value, schema) {
			t.Fatal("unsafe result schema accepted")
		}
	}
	value, _ := Decode([]byte(`{"outcome":"completed","data":{"provider_dispatch":false}}`))
	if !matches(value, schema) {
		t.Fatal("safe exact result rejected")
	}
}

func TestCloudflareInstalledReceiptProcess(t *testing.T) {
	prefix := os.Getenv("SYMPHONY_CLOUDFLARE_TEST_PREFIX")
	if prefix == "" {
		t.Skip("exact installed adapter prefix not selected")
	}
	// Run selects the exact receipt-v2 entrypoint, supplies bounded stdin,
	// captures stdout and validates the exit and full bound result. The owner
	// implementation path is modules/sov-cloudflare-adapter.
	raw, ok, err := Run(context.Background(), prefix, "0.1.0-dev", "capability.list", []byte(`{"store":null,"toolchain_id":null,"query":null}`))
	if err != nil || !ok {
		t.Fatalf("installed process failed: %v %s", err, raw)
	}
	value, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := value.(map[string]any)
	if m["protocol"] != ResultProtocol || m["outcome"] != "completed" || m["effect_state"] != "not_applicable" {
		t.Fatalf("wrong native result %s", raw)
	}
	_, ok, err = Run(context.Background(), prefix, "0.1.0-dev", "capability.list", []byte(`{"store":null,"toolchain_id":null,"query":null,"ambient":"SENTINEL"}`))
	if err != nil || ok {
		t.Fatalf("unknown native input did not reject: %v", err)
	}
	// A copied prefix lets the consumer prove schema tampering rejects before
	// process dispatch without changing the selected qualification installation.
	tmp := t.TempDir()
	err = filepath.WalkDir(prefix, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, e := filepath.Rel(prefix, path)
		if e != nil {
			return e
		}
		dst := filepath.Join(tmp, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0700)
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		return os.WriteFile(dst, b, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
	ownerPath := filepath.Join(tmp, "share/symphony/schemas/sov-cloudflare-adapter/0.1.0-dev/owner.json")
	f, err := os.OpenFile(ownerPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(" ")
	_ = f.Close()
	_, _, err = Run(context.Background(), tmp, "0.1.0-dev", "capability.list", []byte(`{"store":null,"toolchain_id":null,"query":null}`))
	if err == nil || !strings.Contains(err.Error(), "installation unavailable") {
		t.Fatalf("tampered receipt-owned schema accepted: %v", err)
	}
}

func TestCloudflareResourceAndConnectorLostBindings(t *testing.T) {
	for _, op := range []string{"resource.apply", "resource.reconcile", "context.apply", "connector.instance.apply", "connector.installation.reconcile"} {
		request := map[string]any{"operation": op, "context_ref": "sha256:" + strings.Repeat("a", 64), "plan_ref": "sha256:" + strings.Repeat("b", 64)}
		decoded, err := Decode(lost(request))
		if err != nil {
			t.Fatal(err)
		}
		r := decoded.(map[string]any)
		if r["outcome"] != "unknown" || r["effect_state"] != "unresolved" || r["context_ref"] != request["context_ref"] || r["plan_ref"] != request["plan_ref"] {
			t.Fatal("lost mutation bindings", op, r)
		}
	}
}
