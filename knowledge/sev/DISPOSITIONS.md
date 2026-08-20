# SEV Disposition Vocabulary

## Purpose

This file owns the closed v1 disposition vocabulary. A disposition describes the reviewed treatment of one evolution finding. It is not authority to perform the action.

| Disposition | Meaning | Default execution class |
|---|---|---|
| `accept_as_compatible` | Evidence proves no material attunement is required. | report-only |
| `attune_configuration` | A declared extension point can converge through protected configuration. | external apply |
| `extend_contract` | Canonical owner truth must be reviewed and changed. | proposal-only |
| `extend_command_surface` | SSFV/qxctl/operation administration evidence must be added or changed. | proposal-only |
| `install_exact_component` | One exact immutable package is required. | external apply |
| `select_exact_component` | Protected binding must select an installed exact identity. | external apply |
| `dock_exact_component` | Maestro must persist exact receptor presence. | external apply |
| `undock_exact_component` | Maestro must persist exact absence before replacement or removal. | external apply |
| `replace_with_successor` | A declared successor path is required without overwriting history. | composed external apply |
| `preserve_observed_state` | Existing state remains intentionally unchanged and evidenced. | report-only |
| `defer_with_blocker` | Work cannot safely proceed until exact blockers resolve. | report-only |
| `reject_incompatible` | The proposed target contradicts a closed requirement. | report-only |
| `retire_identity` | A stable identity enters terminal lineage without reuse. | proposal plus external apply where applicable |

Unknown dispositions fail closed. A disposition cannot bypass SSIAG, STAV-before-commit, expected-state compare-and-swap, hard dependency order, reobservation, or canonical review.
