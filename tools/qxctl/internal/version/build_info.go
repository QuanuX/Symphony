package version

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
)

const (
	BuildInfoProtocol       = "symphony.qxctl.build-info.v1"
	ReproducibleBuildRecipe = "symphony.qxctl.reproducible-build.v1"
)

// Info describes the running binary. A source stamp is a declaration that the
// build recipe and its retained evidence must verify; it is not release approval.
type Info struct {
	Protocol            string  `json:"protocol"`
	ComponentID         string  `json:"component_id"`
	Version             string  `json:"version"`
	SourceState         string  `json:"source_state"`
	SourceCommit        *string `json:"source_commit"`
	SourceTree          *string `json:"source_tree"`
	SourceDateEpoch     *string `json:"source_date_epoch"`
	DependencyMapSHA256 *string `json:"dependency_map_sha256"`
	BuildRecipe         *string `json:"build_recipe"`
	GoVersion           string  `json:"go_version"`
	GOOS                string  `json:"goos"`
	GOARCH              string  `json:"goarch"`
	ExecutableSHA256    string  `json:"executable_sha256"`
}

type sourceStamp struct {
	commit, tree, epoch, dependencyMap, recipe string
}

var (
	gitObjectPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Pattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Current validates the complete linked source stamp and combines it with the
// actual runtime target/toolchain and the caller's digest of this executable.
func Current(executableSHA256 string) (Info, error) {
	return describe(sourceStamp{SourceCommit, SourceTree, SourceDateEpoch, DependencyMapSHA256, BuildRecipe}, executableSHA256)
}

func describe(stamp sourceStamp, executableSHA256 string) (Info, error) {
	info := Info{
		Protocol: BuildInfoProtocol, ComponentID: "qxctl", Version: Version,
		SourceState: "unavailable", GoVersion: runtime.Version(), GOOS: runtime.GOOS,
		GOARCH: runtime.GOARCH, ExecutableSHA256: executableSHA256,
	}
	if !sha256Pattern.MatchString(executableSHA256) {
		return info, fmt.Errorf("invalid qxctl executable SHA256")
	}
	if stamp == (sourceStamp{}) {
		return info, nil
	}
	info.SourceState = "invalid"
	epoch, err := strconv.ParseInt(stamp.epoch, 10, 64)
	if !gitObjectPattern.MatchString(stamp.commit) || !gitObjectPattern.MatchString(stamp.tree) ||
		!sha256Pattern.MatchString(stamp.dependencyMap) || stamp.recipe != ReproducibleBuildRecipe ||
		err != nil || epoch < 0 || strconv.FormatInt(epoch, 10) != stamp.epoch {
		return info, fmt.Errorf("invalid or incomplete qxctl source stamp")
	}
	info.SourceState = "stamped"
	info.SourceCommit, info.SourceTree, info.SourceDateEpoch = &stamp.commit, &stamp.tree, &stamp.epoch
	info.DependencyMapSHA256, info.BuildRecipe = &stamp.dependencyMap, &stamp.recipe
	return info, nil
}
