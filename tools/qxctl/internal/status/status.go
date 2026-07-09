package status

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/inventory"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/modules"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

type StatusCheck struct {
	Ok bool `json:"ok"`
}

type StatusModules struct {
	Ok    bool `json:"ok"`
	Count int  `json:"count"`
}

type StatusContracts struct {
	Ok    bool `json:"ok"`
	Count int  `json:"count"`
}

type StatusInventory struct {
	Ok     bool   `json:"ok"`
	Schema string `json:"schema"`
}

type StatusDigest struct {
	Ok        bool   `json:"ok"`
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type StatusTool struct {
	Ok        bool   `json:"ok"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Contracts int    `json:"contracts"`
}

type AdministrativeStatus struct {
	Schema     string          `json:"schema"`
	GoBaseline string          `json:"go_baseline"`
	Tool       StatusTool      `json:"tool"`
	Repository StatusCheck     `json:"repository"`
	Modules    StatusModules   `json:"modules"`
	Contracts  StatusContracts `json:"contracts"`
	Inventory  StatusInventory `json:"inventory"`
	Digest     StatusDigest    `json:"digest"`
}

func GetStatus(repoRoot string) (AdministrativeStatus, error) {
	status := AdministrativeStatus{
		Schema:     "qxctl.status.v1",
		GoBaseline: version.GoBaseline,
	}

	// 1. Repository
	if !repository.IsDir(repoRoot) {
		return status, fmt.Errorf("invalid repository root")
	}
	status.Repository.Ok = true

	// 1b. Tool
	qxctlPath := filepath.Join("tools", "qxctl")
	toolContracts := []string{"INTENT.md", "MANIFEST.md", "INSTALL.md", "SKILL.md"}
	
	if !repository.IsDir(filepath.Join(repoRoot, qxctlPath)) {
		return status, fmt.Errorf("missing qxctl directory: %s", qxctlPath)
	}

	for _, file := range toolContracts {
		relPath := filepath.Join(qxctlPath, file)
		if !repository.IsFile(filepath.Join(repoRoot, relPath)) {
			return status, fmt.Errorf("missing tool contract file: %s", relPath)
		}
		meta, err := modules.GetFileMetadataStruct(repoRoot, relPath)
		if err != nil {
			return status, fmt.Errorf("invalid tool contract file: %s", relPath)
		}
		if meta.Lines == 0 || meta.Title == "" {
			return status, fmt.Errorf("invalid tool contract file: %s", relPath)
		}
	}
	status.Tool.Ok = true
	status.Tool.Name = "qxctl"
	status.Tool.Path = "tools/qxctl"
	status.Tool.Contracts = len(toolContracts)

	// 2. Modules
	for _, mod := range modules.CanonicalModules {
		modPath := filepath.Join(repoRoot, "modules", mod)
		if !repository.IsDir(modPath) {
			return status, fmt.Errorf("missing module directory: modules/%s", mod)
		}
	}
	status.Modules.Ok = true
	status.Modules.Count = len(modules.CanonicalModules)

	// 3. Contracts
	contractCount := 0
	for _, mod := range modules.CanonicalModules {
		modRelPath := filepath.Join("modules", mod)
		for _, file := range modules.ExpectedFiles {
			relPath := filepath.Join(modRelPath, file)
			if !repository.IsFile(filepath.Join(repoRoot, relPath)) {
				return status, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
			}
			meta, err := modules.GetFileMetadataStruct(repoRoot, relPath)
			if err != nil {
				return status, fmt.Errorf("invalid contract file: %s in %s", file, modRelPath)
			}
			if meta.Lines == 0 || meta.Title == "" {
				return status, fmt.Errorf("invalid contract file: %s in %s", file, modRelPath)
			}
			contractCount++
		}
	}
	status.Contracts.Ok = true
	status.Contracts.Count = contractCount

	// 4. Inventory
	jsonBytes, err := inventory.SnapshotJSON(repoRoot)
	if err != nil {
		return status, fmt.Errorf("inventory construction failed: %v", err)
	}
	status.Inventory.Ok = true
	status.Inventory.Schema = "qxctl.runtime_inventory.v1"

	// 5. Digest
	hash := sha256.Sum256(jsonBytes)
	hexStr := hex.EncodeToString(hash[:])
	status.Digest.Ok = true
	status.Digest.Algorithm = "sha256"
	status.Digest.Value = hexStr

	return status, nil
}

func Report(repoRoot string) ([]string, error) {
	status, err := GetStatus(repoRoot)
	if err != nil {
		return nil, err
	}

	var output []string
	output = append(output, "status: schema qxctl.status.v1")
	if status.Tool.Ok {
		output = append(output, "status: tool qxctl ok")
	}
	if status.Repository.Ok {
		output = append(output, "status: repository ok")
	}
	if status.Modules.Ok {
		output = append(output, "status: modules ok")
	}
	if status.Contracts.Ok {
		output = append(output, "status: contracts ok")
	}
	if status.Inventory.Ok {
		output = append(output, "status: inventory ok")
	}
	if status.Digest.Ok {
		output = append(output, fmt.Sprintf("status: digest %s %s", status.Digest.Algorithm, status.Digest.Value))
	}
	
	output = append(output, "status: checks passed")
	return output, nil
}

func ReportJSON(repoRoot string) ([]byte, error) {
	status, err := GetStatus(repoRoot)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(status, "", "  ")
}
