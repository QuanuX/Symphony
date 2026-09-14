# SHV PDF adapter manifest

Module `shv-pdf-adapter`, engine `symphony-shv-pdf`, vector `shv`, version `0.2.0-dev`, kind `adapter`. C++26 native executable, statically linked engine foundation, runtime-selected external PDFium C API. Receipt-v2 installs the executable, six owner documents, PDF-ADAPTER.md, schema/templates and licenses. No PDFium binary is bundled or installed.

Version 0.2.0 adds native `graph_project` and `graph_validate`, administered by `qxctl shv pdf graph project|validate|roundtrip`. Project takes an extraction request; validate and roundtrip take `{request,graph}`. Roundtrip additionally selects `--adapter-prefix` and `--adapter-version`. See PDF-ADAPTER.md for exact provenance semantics. Retained 0.1.0 installations remain explicitly selectable for their original operations.
