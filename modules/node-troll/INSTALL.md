# node-troll Install

## Install Status
PROPOSED

## Install Scope
Node-local supervision and daemon operations.

## Supported Installation Modes
- **development**: Source-based build and local test deployment.
- **production**: Bare-metal service installation and lifecycle management.
- **source-evidence / proposal-only state**: Current state. Contract seeds only.

## Configuration Authority
Development config:
```text
${XDG_CONFIG_HOME:-$HOME/.config}/symphony/node-troll/config.yaml
```

Development state:
```text
${XDG_STATE_HOME:-$HOME/.local/state}/symphony/node-troll
```

Production config:
```text
/etc/symphony/node-troll/config.yaml
```

Production state/runtime/logs:
```text
/var/lib/symphony/node-troll
/run/symphony/node-troll
/var/log/symphony/node-troll
```

Legacy paths are explicit migration/compatibility inputs only, not canonical paths:
```text
~/.quanux-node/config.yaml
/etc/quanux-node
/home/quanux/.quanux-node
```

Config precedence:
1. CLI flag
2. environment variable
3. service-managed production path
4. development default
5. explicit legacy migration mode

development must not require root-owned paths.
production must not depend on user-home paths.
legacy QuanuX paths are not canonical Symphony paths.
no implicit legacy fallback.

## Explicit Non-Requirements
- Kubernetes, containers, specific cloud providers.
- Python must not be required for the administrative spine.

## Current Limitations
Proposal-only state. No implementation code or installer scripts exist.

## Ratification Status
DRAFT
