# SNV Common Spec

Mechanics v1: bounded engine-process.v1, strict foundation parser, 1 MiB requests, 4 MiB responses, depth 64, 262144 JSON values, maximum domain profile 2048 records (owners enforce their own fields). Canonical JSON digest binds representation, not factual identity. Errors disclose bounded codes. Owner libraries supply operation contracts and semantic reducers.

After a request envelope is admitted, unexpected C++ exceptions return `internal.failure` with the original request, correlation and operation identifiers and the fixed message `bounded operation failed`. Exception text is never echoed. Before admission, unavailable routing identifiers remain explicit.

Exact installed SDK dependency checks require the receipt's component, release and semantic/neutral identity, unique owned paths, unchanged receipt-owned bytes, configured archives, all exported public files and required CMake configuration/export files. GNUInstallDirs library directory choices remain supported. The bounded ancestor search locates the exact receipt prefix; it does not replace ownership checks. Receipts establish byte correspondence and declared ownership, not publisher authenticity. CMake package configuration has already executed when `find_package` returns, so these checks do not make an untrusted package safe to execute.
