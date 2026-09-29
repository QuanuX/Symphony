package credential

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func useFixture() (UseBinding, time.Time) {
	now := time.Date(2026, 9, 27, 23, 0, 0, 0, time.UTC)
	digest := "sha256:" + strings.Repeat("a", 64)
	return UseBinding{TOPSID: "018f0c3a-7b2d-7e11-8c12-0242ac120002", SubjectID: "fixture-consumer", SubjectKind: "service", SubjectAuthority: "fixture-peer",
		ConsumerDigest: digest, ProviderBindingDigest: digest, PolicyDigest: digest, ConfigDigest: digest, AuthorizationDigest: digest, AuditReceiptDigest: digest,
		LeaseID: "fixture-lease", RequestID: "fixture-request", CorrelationID: "fixture-correlation", Reference: Reference{Provider: "fixture-provider", Name: "fixture-reference", Version: "generation-1", Type: "api-key"},
		Operation: "fixture-use", Resource: "fixture-resource", Audience: "fixture-audience", Scope: "fixture-scope", MaximumBytes: 32, IssuedAt: now, Deadline: now.Add(MaxUseLifetime)}, now
}

func TestUseExactBindingAndOwnership(t *testing.T) {
	b, now := useFixture()
	original := b
	use, status := NewOneShotUse(b, now)
	if status != UseOK {
		t.Fatal(status)
	}
	b.Reference.Version = "generation-2"
	if got := use.Consume(context.Background(), original, now); got != UseOK {
		t.Fatal(got)
	}
	if got := use.Consume(context.Background(), original, now); got != UseSpent {
		t.Fatal(got)
	}
	// The guard has no exported secret/control payload or reconstructible token.
	encoded, err := json.Marshal(use)
	if err != nil || string(encoded) != "{}" {
		t.Fatal("guard serialized internal binding")
	}
	empty := new(OneShotUse)
	if got := empty.Consume(context.Background(), original, now); got == UseOK {
		t.Fatal("zero guard admitted")
	}
}

func TestUseEveryBindingFieldRejectsDrift(t *testing.T) {
	b, now := useFixture()
	// Walk every field so a future binding addition cannot silently omit its drift test.
	v := reflect.ValueOf(b)
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		t.Run(name, func(t *testing.T) {
			current := b
			field := reflect.ValueOf(&current).Elem().Field(i)
			switch field.Kind() {
			case reflect.String:
				switch {
				case name == "TOPSID":
					field.SetString("018f0c3a-7b2d-7e11-8c12-0242ac120003")
				case strings.HasSuffix(name, "Digest"):
					field.SetString("sha256:" + strings.Repeat("b", 64))
				default:
					field.SetString(field.String() + "-changed")
				}
			case reflect.Uint32:
				field.SetUint(field.Uint() + 1)
			case reflect.Struct:
				if field.Type() == reflect.TypeOf(time.Time{}) {
					delta := -time.Nanosecond
					if name == "IssuedAt" {
						delta = time.Nanosecond
					}
					field.Set(reflect.ValueOf(field.Interface().(time.Time).Add(delta)))
				} else {
					ref := field.Interface().(Reference)
					ref.Version = "generation-2"
					field.Set(reflect.ValueOf(ref))
				}
			default:
				t.Fatal("unhandled binding field")
			}
			// Ensure drift checks compare two independently valid bindings, rather
			// than merely exercising syntax rejection.
			checkAt := now.Add(time.Nanosecond)
			if _, status := NewOneShotUse(current, checkAt); status != UseOK {
				t.Fatal("drift fixture is invalid", status)
			}
			use, _ := NewOneShotUse(b, now)
			if got := use.Consume(context.Background(), current, checkAt); got != UseStale {
				t.Fatalf("drift accepted: %s", got)
			}
			if got := use.Consume(context.Background(), b, now); got != UseSpent {
				t.Fatal("drift did not burn use")
			}
		})
	}
	for _, name := range []string{"Provider", "Name", "Version", "Type"} {
		t.Run("reference-"+name, func(t *testing.T) {
			current := b
			f := reflect.ValueOf(&current.Reference).Elem().FieldByName(name)
			f.SetString(f.String() + "-changed")
			u, _ := NewOneShotUse(b, now)
			if u.Consume(context.Background(), current, now) != UseStale {
				t.Fatal("reference drift accepted")
			}
		})
	}
}

func TestUseDeadlineCancellationAndInvalidInput(t *testing.T) {
	b, now := useFixture()
	cases := []struct {
		name string
		edit func(*UseBinding)
	}{
		{"empty-generation", func(b *UseBinding) { b.Reference.Version = "" }},
		{"oversize-token", func(b *UseBinding) { b.RequestID = strings.Repeat("x", 257) }},
		{"control-byte", func(b *UseBinding) { b.Scope = "scope\nvalue" }},
		{"bad-tops", func(b *UseBinding) { b.TOPSID = "../../other" }},
		{"nil-tops", func(b *UseBinding) { b.TOPSID = "00000000-0000-0000-0000-000000000000" }},
		{"uppercase-digest", func(b *UseBinding) { b.ConsumerDigest = "sha256:" + strings.Repeat("A", 64) }},
		{"missing-audit", func(b *UseBinding) { b.AuditReceiptDigest = "" }},
		{"zero-capacity", func(b *UseBinding) { b.MaximumBytes = 0 }},
		{"oversize-capacity", func(b *UseBinding) { b.MaximumBytes = MaxUseBytes + 1 }},
		{"future-issued", func(b *UseBinding) { b.IssuedAt = now.Add(time.Nanosecond) }},
		{"equal-deadline", func(b *UseBinding) { b.Deadline = b.IssuedAt }},
		{"oversize-lifetime", func(b *UseBinding) { b.Deadline = b.Deadline.Add(time.Nanosecond) }},
		{"non-utc", func(b *UseBinding) { b.IssuedAt = b.IssuedAt.In(time.FixedZone("UTC-alias", 0)) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := b
			c.edit(&x)
			u, status := NewOneShotUse(x, now)
			if u != nil || status != UseInvalid {
				t.Fatal("invalid binding admitted")
			}
		})
	}
	if u, status := NewOneShotUse(b, b.Deadline); u != nil || status != UseExpired {
		t.Fatal("expired construction")
	}
	for _, at := range []time.Time{now.Add(-time.Nanosecond), b.Deadline, b.Deadline.Add(time.Nanosecond)} {
		u, _ := NewOneShotUse(b, now)
		if u.Consume(context.Background(), b, at) != UseExpired || u.Consume(context.Background(), b, now) != UseSpent {
			t.Fatal("expiry replay")
		}
	}
	u, _ := NewOneShotUse(b, now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if u.Consume(ctx, b, now) != UseCancelled || u.Consume(context.Background(), b, now) != UseSpent {
		t.Fatal("cancellation replay")
	}
	u, _ = NewOneShotUse(b, now)
	if u.Consume(nil, b, now) != UseInvalid || u.Consume(context.Background(), b, now) != UseSpent {
		t.Fatal("nil context replay")
	}
	var absent *OneShotUse
	if absent.Consume(context.Background(), b, now) != UseInvalid {
		t.Fatal("nil guard")
	}
	u, _ = NewOneShotUse(b, now)
	if u.Consume(context.Background(), b, b.Deadline.Add(-time.Nanosecond)) != UseOK {
		t.Fatal("last valid instant rejected")
	}
}

func TestUseConcurrentSingleConsumption(t *testing.T) {
	b, now := useFixture()
	u, _ := NewOneShotUse(b, now)
	var admitted, spent atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 128; i++ {
		wg.Go(func() {
			switch u.Consume(context.Background(), b, now) {
			case UseOK:
				admitted.Add(1)
			case UseSpent:
				spent.Add(1)
			default:
				t.Error("unexpected state")
			}
		})
	}
	wg.Wait()
	if admitted.Load() != 1 || spent.Load() != 127 {
		t.Fatalf("admitted=%d spent=%d", admitted.Load(), spent.Load())
	}
}
