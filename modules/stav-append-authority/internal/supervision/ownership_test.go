package supervision

import (
	"math"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
)

func TestSupervisorRejectsReservedOwnership(t *testing.T) {
	layout := stavpaths.InstanceLayout{Scope: stavpaths.ScopeSystem, TOPSID: testTOPSID}
	for _, ids := range [][2]uint32{{math.MaxUint32, 20}, {501, math.MaxUint32}} {
		cfg := config.Default(layout, uint64(ids[0]), uint64(ids[1]))
		if _, err := SpecFromConfig(layout.Scope, testTOPSID, "/unused/stav", cfg); err == nil {
			t.Fatal("supervisor admitted reserved config ownership")
		}
		if _, _, err := render(Spec{Scope: layout.Scope, TOPSID: testTOPSID, Binary: "/unused/stav", UID: ids[0], GID: ids[1]}); err == nil || !strings.Contains(err.Error(), "is reserved") {
			t.Fatalf("reserved supervisor identity reached host checks: %v", err)
		}
	}
}
