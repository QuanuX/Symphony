package inventory

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/modules"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

type InventoryModule struct {
	Module    string                     `json:"module"`
	Path      string                     `json:"path"`
	Contracts []modules.ContractMetadata `json:"contracts"`
}

type RuntimeInventory struct {
	Schema        string            `json:"schema"`
	GoBaseline    string            `json:"go_baseline"`
	ModuleCount   int               `json:"module_count"`
	ContractCount int               `json:"contract_count"`
	Modules       []InventoryModule `json:"modules"`
}

// Snapshot generates a plaintext inventory of the first runtime-set module state.
func Snapshot(repoRoot string) ([]string, error) {
	var output []string
	output = append(output, "inventory: schema qxctl.runtime_inventory.v1")

	contractCount := 0

	for _, mod := range modules.CanonicalModules {
		modRelPath := filepath.Join("modules", mod)
		modPath := filepath.Join(repoRoot, modRelPath)

		if !repository.IsDir(modPath) {
			return nil, fmt.Errorf("missing module directory: %s", modRelPath)
		}

		for _, file := range modules.ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if !repository.IsFile(filePath) {
				return nil, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
			}

			meta, err := modules.GetFileMetadataStruct(repoRoot, filepath.Join(modRelPath, file))
			if err != nil {
				return nil, err
			}
			if meta.Lines == 0 || meta.Title == "" {
				return nil, fmt.Errorf("invalid contract file: %s", file)
			}

			contractCount++
		}
		output = append(output, fmt.Sprintf("inventory: module %s ok", mod))
	}

	output = append(output, fmt.Sprintf("inventory: contracts checked %d", contractCount))
	output = append(output, "inventory: checks passed")
	return output, nil
}

// SnapshotJSON generates a JSON inventory of the first runtime-set module state.
func SnapshotJSON(repoRoot string) ([]byte, error) {
	inv := RuntimeInventory{
		Schema:        "qxctl.runtime_inventory.v1",
		GoBaseline:    version.GoBaseline,
		ModuleCount:   len(modules.CanonicalModules),
		ContractCount: 0,
	}

	for _, mod := range modules.CanonicalModules {
		modRelPath := filepath.Join("modules", mod)
		modPath := filepath.Join(repoRoot, modRelPath)

		if !repository.IsDir(modPath) {
			return nil, fmt.Errorf("missing module directory: %s", modRelPath)
		}

		invMod := InventoryModule{
			Module: mod,
			Path:   modRelPath,
		}

		for _, file := range modules.ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			if !repository.IsFile(filePath) {
				return nil, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
			}

			relPath := filepath.Join(modRelPath, file)
			meta, err := modules.GetFileMetadataStruct(repoRoot, relPath)
			if err != nil {
				return nil, err
			}
			if meta.Lines == 0 || meta.Title == "" {
				return nil, fmt.Errorf("invalid contract file: %s", file)
			}

			invMod.Contracts = append(invMod.Contracts, meta)
			inv.ContractCount++
		}

		inv.Modules = append(inv.Modules, invMod)
	}

	return json.MarshalIndent(inv, "", "  ")
}
