package contracts

import (
	"fmt"
	"path/filepath"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
)

var ExpectedModules = []string{
	"modules/node-troll",
	"modules/bus-troll",
	"modules/hotpath-runtime",
}

var ExpectedFiles = []string{
	"INTENT.md",
	"MANIFEST.md",
	"INSTALL.md",
	"SKILL.md",
}

// Verify checks if the required modules and their contract files exist.
func Verify(repoRoot string) ([]string, error) {
	var output []string
	output = append(output, fmt.Sprintf("contracts: repository root %s", repoRoot))

	for _, mod := range ExpectedModules {
		modPath := filepath.Join(repoRoot, mod)
		if !repository.IsDir(modPath) {
			output = append(output, fmt.Sprintf("contracts: missing module %s", mod))
			return output, fmt.Errorf("missing module: %s", mod)
		}
		output = append(output, fmt.Sprintf("contracts: module %s ok", mod))

		for _, file := range ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if !repository.IsFile(filePath) {
				output = append(output, fmt.Sprintf("contracts: missing file %s/%s", mod, file))
				return output, fmt.Errorf("missing file: %s/%s", mod, file)
			}
			output = append(output, fmt.Sprintf("contracts: file %s/%s ok", mod, file))
		}
	}

	output = append(output, "contracts: checks passed")
	return output, nil
}
