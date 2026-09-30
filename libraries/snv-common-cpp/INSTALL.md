# SNV Common Install

Configure this directory using CMake >=3.30 and a C++26 compiler; build, install into a private prefix. Set SYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON to use the exact installed foundation SDK. The uninstall-snv-common-cpp target removes only unchanged receipt-owned files. Shared-root lifecycle ownership requires the existing qxctl lifecycle circuit.
