# CodeQL advanced setup

[The workflow](../workflows/codeql.yml) runs on pull requests targeting `main`,
pushes to `main`, manual dispatch, and Tuesdays at 08:23 UTC. Each language and
platform has its own stable analysis category, so one upload does not replace
another platform's results. All jobs use the `security-extended` query suite.

| Language | Runner | Extraction |
| --- | --- | --- |
| C/C++ | Ubuntu 24.04 and macOS 15 Intel | Source-only (`build-mode: none`) |
| Go | Ubuntu 24.04 and macOS 15 Intel | Explicit build of every module declared in `go.work` |
| Swift | macOS 15 Intel | Release build of `modules/ssiag-provider-macos-keychain` |
| GitHub Actions | Ubuntu 24.04 | Workflow analysis |

## Scope and limits

C/C++ analysis is an initial source-only baseline across Symphony's independent
native projects. CodeQL infers compilation units, include paths, and flags; it
does not reproduce each module's exact CMake configuration or dependency
provenance. Missing external headers, generated files, and inferred macros can
reduce accuracy. Both operating systems are scanned because platform headers
and conditional code differ. CodeQL may exclude third-party and test sources by
its own extraction rules; a successful job does not establish complete coverage.

Symphony's native projects retain their selected C++26 standard. As of this
configuration, CodeQL's published support matrix lists C++23 as beta and does not
claim C++26 support. No project standard is changed for this workflow. Inspect
extraction diagnostics and coverage before relying on an absence of alerts.
A future build-traced native configuration must provision each selected owner's
exact dependencies and account for the modules it covers; building one module
must not be described as scanning every native project.

Go uses the version declared by `go.work`, currently 1.26.5. The module list comes
from that file rather than a duplicated list. The fresh cache and `go build -a`
ensure compilation occurs after CodeQL initialization. Separate Linux and macOS
jobs cover their respective build-tagged implementations. Other operating
systems, custom build tags, and test-only Go code are outside this configuration.

The Swift job compiles the provider without executing it, running its integration
tests, installing it, or accessing Keychain items. The provider and its support
target are Swift; separate C++ test fixtures are outside this Swift build. These
scans are static analysis, not Symphony runtime or release qualification.

## Activation and verification

The repository must have GitHub Actions enabled. Default CodeQL setup must be
disabled when using this advanced workflow; enabling both causes upload conflicts.
No personal access token or custom repository secret is needed: the job grants
`security-events: write` to GitHub's temporary workflow token.

Before merging, check all six pull-request jobs and their uploaded analyses in
**Security → Code scanning**. Review extraction warnings and the tool status page,
including file coverage and analysis categories. A successful scan can still
report vulnerabilities; job success is not a clean security assessment.

After merging, verify the push run on `main` to establish the default-branch
baseline. The weekly schedule and manual dispatch then become available from
`main`. Keep the action commit pins current when updating CodeQL and checkout.
The `setup-go` pin follows the verified v6 release line; its input preserves the
repository's exact Go version.

## References

- [GitHub advanced setup](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/configure-code-scanning/configuring-advanced-setup-for-code-scanning)
- [CodeQL compiled-language build options](https://docs.github.com/en/code-security/reference/code-scanning/codeql/build-options-for-compiled-languages)
- [CodeQL language and compiler support](https://codeql.github.com/docs/codeql-overview/supported-languages-and-frameworks/)
- [CodeQL Action](https://github.com/github/codeql-action)
