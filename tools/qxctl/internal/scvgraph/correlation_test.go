package scvgraph

import (
	"bytes"
	"encoding/json"
	"errors"
	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func correlationGraphPath(t *testing.T, s Store) string {
	t.Helper()
	var found string
	if err := filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "state.json" {
			found = path
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if found == "" {
		t.Fatal("graph state absent")
	}
	return found
}
func correlationGraphRead(t *testing.T, s Store) []byte {
	t.Helper()
	raw, err := os.ReadFile(correlationGraphPath(t, s))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func correlationGraphWrite(t *testing.T, s Store, d Document) []byte {
	t.Helper()
	var err error
	d.Digest, err = selfDigest(d)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := knowledgeengine.SCVCanonical(d)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(correlationGraphPath(t, s), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return raw
}
func correlationGraphLegacy(t *testing.T, s Store) Document {
	t.Helper()
	var d Document
	if err := json.Unmarshal(correlationGraphRead(t, s), &d); err != nil {
		t.Fatal(err)
	}
	d.Protocol = legacyProtocol
	for id, a := range d.Operations {
		a.CorrelationID = ""
		d.Operations[id] = a
	}
	correlationGraphWrite(t, s, d)
	return d
}

func TestGraphCorrelationDurableBeforeCallbackAndStableAcrossRecovery(t *testing.T) {
	s, installation := fixture(t)
	i := intent(t, "opaque-selection", nil, 0, graph(t, "fixture"), installation)
	var first string
	if _, err := s.Select(i, func(got Intent, correlation string) (Authorization, error) {
		first = correlation
		if stavprotocol.ValidateRequestUUID(correlation) != nil || got.Digest != i.Digest || correlation == i.OperationID {
			t.Fatal("correlation changed native intent or is not a UUID")
		}
		var d Document
		if err := json.Unmarshal(correlationGraphRead(t, s), &d); err != nil {
			t.Fatal(err)
		}
		if d.Protocol != protocol || d.Operations[i.OperationID].CorrelationID != correlation || d.Operations[i.OperationID].Status != "prepared" || d.GraphDigest != nil {
			t.Fatal("callback ran before durable correlation")
		}
		return Authorization{}, errors.New("fixture denial")
	}); err == nil {
		t.Fatal("fixture denial ignored")
	}
	if _, err := s.Select(i, func(got Intent, correlation string) (Authorization, error) {
		if correlation != first || got.Digest != i.Digest {
			t.Fatal("recovery rebound intent/correlation")
		}
		return Authorization{Evidence: json.RawMessage(`{"fixture":"expired"}`), ValidUntil: time.Now().Add(-time.Second)}, nil
	}); err == nil {
		t.Fatal("expired authorization committed")
	}
	d, err := s.Select(i, func(got Intent, correlation string) (Authorization, error) {
		if correlation != first {
			t.Fatal("expired publication lost correlation")
		}
		return authority(got, correlation)
	})
	if err != nil || d.Generation != 1 || d.Operations[i.OperationID].CorrelationID != first || len(d.Operations[i.OperationID].Authorizations) != 2 {
		t.Fatal("stable recovery failed", err)
	}
}

func TestGraphCorrelationLegacyPendingMigrationPreservesIntentAndHistory(t *testing.T) {
	for _, id := range []string{"opaque-selection", "d03594fd-9d9c-42b8-9471-a32eb2242271", "0199d5e1-9000-7001-8001-a32eb2242271"} {
		t.Run(id, func(t *testing.T) {
			s, installation := fixture(t)
			i := intent(t, id, nil, 0, graph(t, "fixture"), installation)
			if _, err := s.Select(i, nil); err == nil {
				t.Fatal("missing authority accepted")
			}
			d := correlationGraphLegacy(t, s)
			a := d.Operations[id]
			a.Status = "authorized"
			a.Authorizations = []json.RawMessage{json.RawMessage(`{"legacy_fixture":"untouched"}`)}
			d.Operations[id] = a
			before := correlationGraphWrite(t, s, d)
			if observed, err := s.Inspect(); err != nil || observed.Protocol != legacyProtocol {
				t.Fatal("legacy inspection", err)
			}
			if !bytes.Equal(before, correlationGraphRead(t, s)) {
				t.Fatal("inspection migrated journal")
			}
			if _, err := s.Select(i, func(got Intent, correlation string) (Authorization, error) {
				var retained Document
				if err := json.Unmarshal(correlationGraphRead(t, s), &retained); err != nil {
					t.Fatal(err)
				}
				after := retained.Operations[id]
				if retained.Protocol != protocol || got.Digest != a.Intent.Digest || after.Status != a.Status || !bytes.Equal(after.Authorizations[0], a.Authorizations[0]) || after.CorrelationID != correlation || stavprotocol.ValidateRequestUUID(correlation) != nil {
					t.Fatal("migration rewrote native intent/history")
				}
				if stavprotocol.ValidateRequestUUID(id) == nil && correlation != id {
					t.Fatal("eligible legacy UUID correlation changed")
				}
				return Authorization{}, errors.New("stop after durable migration")
			}); err == nil {
				t.Fatal("fixture denial ignored")
			}
		})
	}
}

func TestGraphCorrelationLegacyCommittedReplayAndStalePendingDoNotWrite(t *testing.T) {
	s, installation := fixture(t)
	pending := intent(t, "pending", nil, 0, graph(t, "pending"), installation)
	committed := intent(t, "committed", nil, 0, graph(t, "committed"), installation)
	if _, err := s.Select(pending, nil); err == nil {
		t.Fatal("missing authority accepted")
	}
	if _, err := s.Select(committed, authority); err != nil {
		t.Fatal(err)
	}
	correlationGraphLegacy(t, s)
	before := correlationGraphRead(t, s)
	unexpected := func(Intent, string) (Authorization, error) {
		t.Fatal("read/rejected stale state requested authority")
		return Authorization{}, nil
	}
	if d, err := s.Select(committed, unexpected); err != nil || d.Protocol != legacyProtocol {
		t.Fatal("committed legacy replay", err)
	}
	if _, err := s.Select(pending, unexpected); err == nil {
		t.Fatal("stale pending intent migrated")
	}
	if !bytes.Equal(before, correlationGraphRead(t, s)) {
		t.Fatal("committed replay/stale rejection changed legacy bytes")
	}
}

func TestGraphCorrelationRejectsInvalidUUIDAndLegacyField(t *testing.T) {
	for _, value := range []string{"opaque", "00000000-0000-0000-0000-000000000000", "d03594fd-9d9c-52b8-9471-a32eb2242271", "legacy-valid"} {
		t.Run(value, func(t *testing.T) {
			s, installation := fixture(t)
			i := intent(t, "one", nil, 0, graph(t, "one"), installation)
			if _, err := s.Select(i, nil); err == nil {
				t.Fatal("missing authority accepted")
			}
			var d Document
			if err := json.Unmarshal(correlationGraphRead(t, s), &d); err != nil {
				t.Fatal(err)
			}
			a := d.Operations[i.OperationID]
			a.CorrelationID = value
			if value == "legacy-valid" {
				d.Protocol = legacyProtocol
				a.CorrelationID = "d03594fd-9d9c-42b8-9471-a32eb2242271"
			}
			d.Operations[i.OperationID] = a
			correlationGraphWrite(t, s, d)
			if _, err := s.Inspect(); err == nil {
				t.Fatal("invalid correlation accepted under resealed journal")
			}
		})
	}
}
