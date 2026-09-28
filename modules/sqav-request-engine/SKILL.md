# SQAV request engine usage contract

Choose an explicit installed prefix and release. Discover one of the three request variants with `qxctl sqv schema --operation request_validate --adapter NAME --version 0.1.0-dev --prefix PREFIX --json` or an unanswered template with `qxctl sqv template` using the same selection flags. Supply all required fields, then run `qxctl sqv acquisition validate --input FILE --prefix PREFIX --version 0.1.0-dev --json`. The response establishes native request validity only. Read SPEC.md before interpreting the result. No credential field is admitted.
