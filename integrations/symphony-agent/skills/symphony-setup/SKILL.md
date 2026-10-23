---
name: symphony-setup
description: Set up or repair this plugin's local Symphony connection by selecting an existing qxctl executable, creating a private pinned connection profile, and verifying read-only discovery through the host.
---

# Set up Symphony

Connect the installed plugin to the Symphony installation the user selected. The bridge is a local executable; setup does not install Symphony or establish a remote service.

1. Locate this plugin's root and host-provided persistent plugin data directory. Read [the connection profile contract](../../docs/connection-profile.md). Resolve the selected qxctl executable and working directory from the user's request or existing installation evidence. If several installations are plausible, ask which one to use before pinning one. Do not select a different version to make a failing check pass.
2. Check that the packaged binary matches the executing operating system and architecture. Read [host compatibility](../../docs/host-compatibility.md) for standalone or Windows-to-WSL configuration. Linux and macOS binaries are distinct; WSL launches the Linux binary inside the chosen distribution. DGX/Spark hardware qualification is deferred.
3. For the packaged OpenAI plugin, its MCP entry reads `${PLUGIN_DATA}/connection.json`. Use the actual host-resolved data directory, not the literal placeholder or a guessed home-directory path. Create its parent privately if needed. If that location is unavailable, generate a standalone Codex configuration fragment targeting an explicit private profile, explain the selected mode, and preserve existing host settings.
4. Run the packaged `scripts/configure_connection.py` with `--qxctl`, `--working-directory`, and `--output` using those exact absolute paths. The helper requires Python 3.11 or newer and creates the profile exclusively with mode 0600. Start with the default read-only profile. Never overwrite an existing profile blindly: inspect its nonsecret selection and preserve it before a user-requested replacement. Do not print credential-bearing environment values.
5. Reload/reconnect the Symphony MCP connection using the host's supported mechanism. Call `symphony_connection`, then `symphony_discover` and a suitable `symphony_describe`. If an existing repository inspection is appropriate, perform it through `symphony_read`. Report the observed executable/registry binding and available commands; discovery does not prove engines or live services are healthy.

A missing connection profile can make the MCP server fail at first launch; the packaged setup skill remains the route to configuring it. If the host requires a new session or restart, state that remaining step accurately. Do not claim a connection merely because a configuration file was written.

Only enable specific non-read command IDs when the user has requested the corresponding administration and the installation's native authority is available. Setup neither grants native permissions nor authorizes arbitrary later mutations. Use the administration skill for an actual operation and preserve owner refusal/recovery semantics.
