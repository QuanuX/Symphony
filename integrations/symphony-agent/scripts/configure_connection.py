#!/usr/bin/env python3
"""Create a protected, exact-version Symphony bridge profile without installing it."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys

def sha256(path):
    with open(path, 'rb') as stream:
        return 'sha256:' + hashlib.file_digest(stream, 'sha256').hexdigest()

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--qxctl', required=True, type=Path)
    parser.add_argument('--working-directory', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--enable', action='append', default=[], metavar='EXACT_COMMAND_ID', help='Enable one non-read operation; repeat for more')
    parser.add_argument('--inherit', action='append', default=[], metavar='ENV_NAME', help='Explicit additional environment name inherited by qxctl')
    parser.add_argument('--timeout-ms', type=int, default=30000)
    parser.add_argument('--max-output-bytes', type=int, default=524288)
    args = parser.parse_args()
    qxctl = args.qxctl.resolve(strict=True)
    cwd = args.working_directory.resolve(strict=True)
    if not qxctl.is_file() or not cwd.is_dir():
        parser.error('Select an executable file and existing working directory')
    if not 100 <= args.timeout_ms <= 60000 or not 1024 <= args.max_output_bytes <= 8*1024*1024:
        parser.error('Process limits are outside the supported bounds')
    names = list(dict.fromkeys(['HOME', 'TMPDIR'] + args.inherit))
    if any(not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_]*', name) for name in names):
        parser.error('Invalid environment name')
    env = {'PATH': '/usr/bin:/bin:/usr/sbin:/sbin', 'LANG': 'C.UTF-8'}
    env.update({name: os.environ[name] for name in names if name in os.environ})
    before = sha256(qxctl)
    # This is the selected trusted local executable, not a model-supplied shell command.
    completed = subprocess.run([str(qxctl), 'commands', 'manifest', '--json'], cwd=cwd, env=env, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=args.timeout_ms/1000, check=False)
    if completed.returncode or len(completed.stdout) > 8*1024*1024:
        parser.error('Selected qxctl did not provide a successful bounded manifest; no profile written')
    manifest = json.loads(completed.stdout)
    if sha256(qxctl) != before or manifest.get('executable_digest') != before or manifest.get('protocol') != 'symphony.qxctl.command-registry.v1':
        parser.error('Executable/manifest binding failed')
    digest = manifest.get('registry_digest', '')
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', digest):
        parser.error('Invalid manifest registry digest')
    records = {c['command_id']: c for c in manifest['commands']}
    for enabled in args.enable:
        record = records.get(enabled, {})
        if record.get('mutability') not in ('permission_backed_mutation', 'evidence_only', 'proposal_only') or record.get('visibility') != 'public' or not record.get('noninteractive'):
            parser.error('Enabled ID must be a public, noninteractive, non-read command in this exact registry: '+enabled)
    config = dict(schema='symphony.agent.connection.v1', qxctl=str(qxctl), qxctl_sha256=before, working_directory=str(cwd), registry_digest=digest, enabled_nonread_commands=sorted(set(args.enable)), inherit_environment=names, environment={}, timeout_ms=args.timeout_ms, max_output_bytes=args.max_output_bytes)
    # Exclusive creation: never overwrite a user's existing connection profile.
    fd = os.open(args.output, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as stream:
        json.dump(config, stream, indent=2)
        stream.write('\n')
    print(json.dumps({'profile': str(args.output.resolve()), 'qxctl_sha256': before, 'registered_commands': len(records), 'enabled_nonread_commands': config['enabled_nonread_commands']}))

if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        print('Cannot configure connection: '+str(error), file=sys.stderr)
        sys.exit(2)
