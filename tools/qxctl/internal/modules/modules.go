package modules

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
)

var CanonicalModules = []string{
	"node-troll",
	"bus-troll",
	"hotpath-runtime",
}

var ExpectedFiles = []string{
	"INTENT.md",
	"MANIFEST.md",
	"INSTALL.md",
	"SKILL.md",
}

type ContractMetadata struct {
	Path   string `json:"path"`
	Title  string `json:"title"`
	Bytes  int64  `json:"bytes"`
	Lines  int    `json:"lines"`
	Sha256 string `json:"sha256"`
}

type ModuleMetadata struct {
	Schema    string             `json:"schema,omitempty"`
	Module    string             `json:"module"`
	Contracts []ContractMetadata `json:"contracts"`
}

type ModulesMetadata struct {
	Schema  string           `json:"schema"`
	Modules []ModuleMetadata `json:"modules"`
}

// List verifies the presence of all canonical modules and returns their status.
func List(repoRoot string) ([]string, error) {
	var output []string
	output = append(output, fmt.Sprintf("modules: repository root %s", repoRoot))

	for _, mod := range CanonicalModules {
		modPath := filepath.Join(repoRoot, "modules", mod)
		if !repository.IsDir(modPath) {
			output = append(output, fmt.Sprintf("modules: missing module modules/%s", mod))
			return output, fmt.Errorf("missing required module directory: modules/%s", mod)
		}
		output = append(output, fmt.Sprintf("modules: %s modules/%s ok", mod, mod))
	}

	output = append(output, "modules: checks passed")
	return output, nil
}

// Inspect reads the contract files for a given canonical module.
func Inspect(repoRoot, moduleName string) ([]string, error) {
	// Verify it's a known canonical module
	isKnown := false
	for _, mod := range CanonicalModules {
		if mod == moduleName {
			isKnown = true
			break
		}
	}

	if !isKnown {
		return nil, fmt.Errorf("unknown module: %s", moduleName)
	}

	var output []string
	output = append(output, fmt.Sprintf("module: %s", moduleName))

	modRelPath := filepath.Join("modules", moduleName)
	modPath := filepath.Join(repoRoot, modRelPath)

	output = append(output, fmt.Sprintf("path: %s", modRelPath))

	if !repository.IsDir(modPath) {
		return output, fmt.Errorf("missing module directory: %s", modRelPath)
	}

	for _, file := range ExpectedFiles {
		filePath := filepath.Join(modPath, file)
		if !repository.IsFile(filePath) {
			output = append(output, fmt.Sprintf("contract: %s missing", file))
			return output, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
		}
		output = append(output, fmt.Sprintf("contract: %s ok", file))

		title, err := extractH1(filePath)
		if err == nil && title != "" {
			output = append(output, fmt.Sprintf("title: %s %s", file, title))
		}
	}

	output = append(output, "inspection: checks passed")
	return output, nil
}

// Check validates the shape of contract files for a given canonical module.
func Check(repoRoot, moduleName string) ([]string, error) {
	isKnown := false
	for _, mod := range CanonicalModules {
		if mod == moduleName {
			isKnown = true
			break
		}
	}

	if !isKnown {
		return nil, fmt.Errorf("unknown module: %s", moduleName)
	}

	var output []string
	output = append(output, fmt.Sprintf("module check: %s", moduleName))

	modRelPath := filepath.Join("modules", moduleName)
	modPath := filepath.Join(repoRoot, modRelPath)

	if !repository.IsDir(modPath) {
		output = append(output, fmt.Sprintf("module check: missing directory %s", modRelPath))
		return output, fmt.Errorf("missing module directory: %s", modRelPath)
	}

	for _, file := range ExpectedFiles {
		filePath := filepath.Join(modPath, file)
		if !repository.IsFile(filePath) {
			output = append(output, fmt.Sprintf("module check: missing contract %s", file))
			return output, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
		}

		info, err := os.Stat(filePath)
		if err != nil {
			output = append(output, fmt.Sprintf("module check: error reading %s", file))
			return output, fmt.Errorf("error reading %s: %w", file, err)
		}
		if info.Size() == 0 {
			output = append(output, fmt.Sprintf("module check: empty contract %s", file))
			return output, fmt.Errorf("empty contract file: %s", file)
		}

		title, err := extractH1(filePath)
		if err != nil || title == "" {
			output = append(output, fmt.Sprintf("module check: missing H1 in %s", file))
			return output, fmt.Errorf("missing H1 in %s", file)
		}

		output = append(output, fmt.Sprintf("module check: contract %s ok", file))
	}

	output = append(output, "module check: checks passed")
	return output, nil
}

// CheckAll validates the shape of contract files for all canonical modules.
func CheckAll(repoRoot string) ([]string, error) {
	var output []string

	for _, mod := range CanonicalModules {
		_, err := Check(repoRoot, mod)
		if err != nil {
			output = append(output, fmt.Sprintf("modules check: %s failed", mod))
			return output, err
		}
		output = append(output, fmt.Sprintf("modules check: %s ok", mod))
	}

	output = append(output, "modules check: checks passed")
	return output, nil
}

// Metadata extracts deterministic metadata from the contract files for a module.
func Metadata(repoRoot, moduleName string) ([]string, error) {
	isKnown := false
	for _, mod := range CanonicalModules {
		if mod == moduleName {
			isKnown = true
			break
		}
	}

	if !isKnown {
		return nil, fmt.Errorf("unknown module: %s", moduleName)
	}

	var output []string
	output = append(output, fmt.Sprintf("module metadata: %s", moduleName))

	modRelPath := filepath.Join("modules", moduleName)
	modPath := filepath.Join(repoRoot, modRelPath)

	if !repository.IsDir(modPath) {
		output = append(output, fmt.Sprintf("module metadata: missing directory %s", modRelPath))
		return output, fmt.Errorf("missing module directory: %s", modRelPath)
	}

	for _, file := range ExpectedFiles {
		filePath := filepath.Join(modPath, file)
		relPath := filepath.Join(modRelPath, file)
		
		if !repository.IsFile(filePath) {
			output = append(output, fmt.Sprintf("metadata: missing contract %s", relPath))
			return output, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
		}

		meta, err := getFileMetadata(repoRoot, relPath)
		if err != nil {
			output = append(output, fmt.Sprintf("metadata: error reading %s: %v", relPath, err))
			return output, err
		}
		output = append(output, meta...)
	}

	output = append(output, "module metadata: checks passed")
	return output, nil
}

// MetadataAll validates the metadata for all canonical modules.
func MetadataAll(repoRoot string) ([]string, error) {
	var output []string

	for _, mod := range CanonicalModules {
		_, err := Metadata(repoRoot, mod)
		if err != nil {
			output = append(output, fmt.Sprintf("modules metadata: %s failed", mod))
			return output, err
		}
		output = append(output, fmt.Sprintf("modules metadata: %s ok", mod))
	}

	output = append(output, "modules metadata: checks passed")
	return output, nil
}

func MetadataJSON(repoRoot, moduleName string) ([]byte, error) {
	isKnown := false
	for _, mod := range CanonicalModules {
		if mod == moduleName {
			isKnown = true
			break
		}
	}

	if !isKnown {
		return nil, fmt.Errorf("unknown module: %s", moduleName)
	}

	modRelPath := filepath.Join("modules", moduleName)
	modPath := filepath.Join(repoRoot, modRelPath)

	if !repository.IsDir(modPath) {
		return nil, fmt.Errorf("missing module directory: %s", modRelPath)
	}

	moduleMeta := ModuleMetadata{
		Schema: "qxctl.contract_metadata.v1",
		Module: moduleName,
	}

	for _, file := range ExpectedFiles {
		filePath := filepath.Join(modPath, file)
		relPath := filepath.Join(modRelPath, file)
		
		if !repository.IsFile(filePath) {
			return nil, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
		}

		meta, err := getFileMetadataStruct(repoRoot, relPath)
		if err != nil {
			return nil, err
		}
		moduleMeta.Contracts = append(moduleMeta.Contracts, meta)
	}

	return json.MarshalIndent(moduleMeta, "", "  ")
}

func MetadataAllJSON(repoRoot string) ([]byte, error) {
	modulesMeta := ModulesMetadata{
		Schema: "qxctl.modules_contract_metadata.v1",
	}

	for _, mod := range CanonicalModules {
		modRelPath := filepath.Join("modules", mod)
		modPath := filepath.Join(repoRoot, modRelPath)

		if !repository.IsDir(modPath) {
			return nil, fmt.Errorf("missing module directory: %s", modRelPath)
		}

		moduleMeta := ModuleMetadata{
			Module: mod,
		}

		for _, file := range ExpectedFiles {
			filePath := filepath.Join(modPath, file)
			relPath := filepath.Join(modRelPath, file)
			
			if !repository.IsFile(filePath) {
				return nil, fmt.Errorf("missing contract file: %s in %s", file, modRelPath)
			}

			meta, err := getFileMetadataStruct(repoRoot, relPath)
			if err != nil {
				return nil, err
			}
			moduleMeta.Contracts = append(moduleMeta.Contracts, meta)
		}

		modulesMeta.Modules = append(modulesMeta.Modules, moduleMeta)
	}

	return json.MarshalIndent(modulesMeta, "", "  ")
}

func getFileMetadata(repoRoot, relPath string) ([]string, error) {
	absPath := filepath.Join(repoRoot, relPath)
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("empty contract file: %s", relPath)
	}
	byteCount := info.Size()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}
	hashHex := hex.EncodeToString(hasher.Sum(nil))

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	var firstH1 string
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		if firstH1 == "" && strings.HasPrefix(line, "# ") {
			firstH1 = line
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if firstH1 == "" {
		return nil, fmt.Errorf("missing H1 in %s", relPath)
	}

	var output []string
	output = append(output, fmt.Sprintf("metadata: %s", relPath))
	output = append(output, fmt.Sprintf("title: %s", firstH1))
	output = append(output, fmt.Sprintf("bytes: %d", byteCount))
	output = append(output, fmt.Sprintf("lines: %d", lineCount))
	output = append(output, fmt.Sprintf("sha256: %s", hashHex))

	return output, nil
}

func getFileMetadataStruct(repoRoot, relPath string) (ContractMetadata, error) {
	var meta ContractMetadata
	meta.Path = relPath

	absPath := filepath.Join(repoRoot, relPath)
	file, err := os.Open(absPath)
	if err != nil {
		return meta, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return meta, err
	}
	if info.Size() == 0 {
		return meta, fmt.Errorf("empty contract file: %s", relPath)
	}
	meta.Bytes = info.Size()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return meta, err
	}
	meta.Sha256 = hex.EncodeToString(hasher.Sum(nil))

	if _, err := file.Seek(0, 0); err != nil {
		return meta, err
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		if meta.Title == "" && strings.HasPrefix(line, "# ") {
			meta.Title = line
		}
	}
	if err := scanner.Err(); err != nil {
		return meta, err
	}

	if meta.Title == "" {
		return meta, fmt.Errorf("missing H1 in %s", relPath)
	}
	meta.Lines = lineCount

	return meta, nil
}

func extractH1(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			return line, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}
