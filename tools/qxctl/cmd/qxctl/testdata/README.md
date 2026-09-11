# Retained CLI fixture provenance

`help.golden` preserves the earlier static-help fixture because historical SCLV records reference this exact path. It is no longer consumed by help tests or runtime help. Current help is derived from the registered Cobra tree and checked by `cli_help_test.go`.
