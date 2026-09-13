package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvworkflow"
)

// This opt-in benchmark reads an explicitly selected, retained owner record.
// It measures local parsing/validation/projection, never installation checking
// or owner execution. Fixture setup and the output oracle are outside timers.
type scvProjectionBenchmarkFixture struct {
	record       scvworkflow.Record
	logical      string
	inputBundle  json.RawMessage
	resultBundle json.RawMessage
	projection   logicalArtifactReference
	oracle       json.RawMessage
	oracleSHA256 string
}

func scvProjectionBenchmarkInput(t testing.TB) scvProjectionBenchmarkFixture {
	t.Helper()
	var fixture scvProjectionBenchmarkFixture
	path := os.Getenv("SYMPHONY_SCV_PROJECTION_RECORD")
	if path == "" {
		t.Skip("requires SYMPHONY_SCV_PROJECTION_RECORD pointing to an exact retained bundle record")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(io.LimitReader(f, scvworkflow.MaxBytes+1))
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if len(raw) > scvworkflow.MaxBytes {
		t.Fatal("benchmark record exceeds existing four-MiB bound")
	}
	if err = knowledgeengine.ValidateSCVBundleUnicode(raw); err != nil {
		t.Fatal(err)
	}
	fixture.record, err = scvworkflow.ReadRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.record.Operation != "composition_bundle_evaluate" {
		t.Fatal("benchmark requires an actual retained composition_bundle_evaluate record")
	}
	input, err := scvworkflow.Decode(fixture.record.Input)
	if err != nil {
		t.Fatal(err)
	}
	result, err := scvworkflow.Decode(fixture.record.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	fixture.logical, _ = input["operation"].(string)
	fixture.inputBundle, err = knowledgeengine.SCVCanonical(input["bundle"])
	if err != nil {
		t.Fatal(err)
	}
	fixture.resultBundle, err = knowledgeengine.SCVCanonical(result["result_bundle"])
	if err != nil {
		t.Fatal(err)
	}
	fixture.projection, err = projectLogicalRecord(fixture.record, fixture.logical)
	if err != nil {
		t.Fatal(err)
	}
	fixture.oracle, err = knowledgeengine.SCVCanonical(map[string]any{
		"protocol":   "local.scv.logical-projection-benchmark-oracle.v1",
		"record_ref": fixture.record.Digest, "installation": fixture.record.Installation,
		"logical_input": fixture.projection.Input, "logical_result": fixture.projection.Artifact,
		"descriptor": fixture.projection.Descriptor,
	})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(fixture.oracle)
	fixture.oracleSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	if expected := os.Getenv("SYMPHONY_SCV_PROJECTION_EXPECTED_SHA256"); expected != "" && expected != fixture.oracleSHA256 {
		t.Fatalf("canonical projection changed: got %s, expected %s", fixture.oracleSHA256, expected)
	}
	return fixture
}

// Run separately from timed samples if an exact portable baseline is wanted.
// Exclusive creation prevents a later candidate run replacing its own oracle.
func TestSCVLogicalProjectionOracle(t *testing.T) {
	fixture := scvProjectionBenchmarkInput(t)
	if path := os.Getenv("SYMPHONY_SCV_PROJECTION_ORACLE_OUT"); path != "" {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Write(fixture.oracle)
		closeErr := f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	t.Logf("oracle=%s record=%s operation=%s input_bytes=%d result_bytes=%d", fixture.oracleSHA256, fixture.record.Digest, fixture.logical, len(fixture.projection.Input), len(fixture.projection.Artifact))
}

var scvProjectionBenchmarkSink logicalArtifactReference
var scvProjectionBytesSink json.RawMessage

func BenchmarkSCVProjectLogicalRecord(b *testing.B) {
	fixture := scvProjectionBenchmarkInput(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(fixture.record.Input) + len(fixture.record.Artifact)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		projection, err := projectLogicalRecord(fixture.record, fixture.logical)
		if err != nil {
			b.Fatal(err)
		}
		scvProjectionBenchmarkSink = projection
	}
}

func BenchmarkSCVProjectionValidateResult(b *testing.B) {
	fixture := scvProjectionBenchmarkInput(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(fixture.record.Input) + len(fixture.record.Artifact)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := knowledgeengine.ValidateSCVResult(fixture.record.Operation, fixture.record.Input, fixture.record.Artifact); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSCVProjectionBundleDecode(b *testing.B) {
	fixture := scvProjectionBenchmarkInput(b)
	for _, item := range []struct {
		name string
		raw  json.RawMessage
	}{{"input", fixture.inputBundle}, {"result", fixture.resultBundle}} {
		b.Run(item.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(item.raw)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				raw, err := knowledgeengine.SCVBundleDecode(item.raw)
				if err != nil {
					b.Fatal(fmt.Errorf("%s bundle: %w", item.name, err))
				}
				scvProjectionBytesSink = raw
			}
		})
	}
}
