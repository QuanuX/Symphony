package supervision

import (
	"math"
	"strings"
	"testing"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
)

func TestSupervisorRejectsReservedOwnership(t *testing.T) {
	topsID := "01234567-89ab-4def-8123-456789abcdef"
	layout := ssiagpaths.InstanceLayout{Scope: ssiagpaths.ScopeSystem, TOPSID: topsID}
	for _, ids := range [][2]uint32{{math.MaxUint32, 20}, {501, math.MaxUint32}} {
		cfg := config.Default(layout, "Desk", &ids[0], &ids[1])
		if _, err := SpecFromConfig(layout.Scope, topsID, "/unused/ssiag", cfg); err == nil {
			t.Fatal("supervisor admitted reserved config ownership")
		}
		if _, _, err := render(Spec{Scope: layout.Scope, TOPSID: topsID, Binary: "/unused/ssiag", UID: ids[0], GID: ids[1]}); err == nil || !strings.Contains(err.Error(), "is reserved") {
			t.Fatalf("reserved supervisor identity reached host checks: %v", err)
		}
	}
}
