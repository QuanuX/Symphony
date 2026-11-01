package version

// Version remains dev for ordinary local builds. An explicit candidate build
// supplies its independent package identity through Go's linker -X option.
var Version = "dev"
