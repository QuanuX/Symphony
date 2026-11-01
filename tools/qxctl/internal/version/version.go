package version

// Version is the current version of the qxctl command surface.
const Version = "0.1.0-dev"

// GoBaseline is the current scripted baseline target.
const GoBaseline = "go1.26.5"

// Source stamps are supplied together by the reproducible build recipe. An
// ordinary go build leaves them empty and makes no source-binding assertion.
var (
	SourceCommit        string
	SourceTree          string
	SourceDateEpoch     string
	DependencyMapSHA256 string
	BuildRecipe         string
)
