package modules

import (
	"bufio"
	"fmt"
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
