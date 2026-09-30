# SNV Common Install

Configure this directory using CMake >=3.30 and a C++26 compiler; build, install into a private prefix. Set SYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON to use the exact installed foundation SDK. The uninstall-snv-common-cpp target removes only unchanged receipt-owned files. Shared-root lifecycle ownership requires the existing qxctl lifecycle circuit.

The native `snv-common-cpp-tests` target checks failure envelopes without an installation. For an existing exact installed prefix, build and run `snv-sdk-receipt-admission-tests` with `--source`, `--prefix` and `--compiler` (optional `--output` saves exact results). It copies the supplied prefix into fresh private cases and checks valid admission and omitted-header, omitted-CMake-file and duplicate-owned-path refusals. The supplied prefix is preserved.
