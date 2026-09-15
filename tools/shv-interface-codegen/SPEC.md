# SHV interface authoring specification

This bounded Python authoring tool projects source and publication owner declarations into C++ descriptor metadata, Go release admission and CMake package inventories. It does not generate semantic handlers, authority policies, storage operations or user requirements. Python 3.9+ and gofmt are required for generation; checked-in outputs build without Python when testing is disabled.

Each module owns OWNER-INTERFACE.json. Frozen installed descriptors and a separate history digest lock preserve historical operations. Duplicate JSON keys, missing or unknown fields, changed historical operations, unknown owners, changed embedded dependencies, incomplete schema inventories, nonlocal/unresolved schema references and unsafe file paths fail generation. The current bounded migration permits metadata-only releases with the same operations as the latest frozen release. Extending that authoring scope requires an explicit owner-contract change; this is not a restriction on user-created modules.

Run `python3 tools/shv-interface-codegen/generate.py --owner shv-source-engine` (or `shv-publication-engine`); add `--check` to verify checked-in output without writing it. CTest runs both frozen native descriptor parity and negative generator tests. Generated C++ retains all administrative interactions, including publication apply and recover.

The new package owns its declaration through receipt-v2. qxctl compares its canonical declaration digest with compiled admission after verifying receipt/file ownership. Resealing a changed declaration does not admit a new interface. Semantic validators independently replay the selected owner's results. Existing command families and defaults remain unchanged.

Administration disposition: repository authoring/build support; no runtime qxctl leaf is added. Runtime source operations remain under `qxctl shv source`; publication remains under `qxctl shv catalogue publication`. This tool neither installs nor invokes either owner.
