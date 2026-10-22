#!/usr/bin/env python3
"""Create only an isolated qxctl fixture for the Windows-to-WSL stdio check."""
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys

root=Path('/work/wsl-fixture')
root.mkdir(mode=0o700,exist_ok=False)
sys.argv=['fixture','/work/result/build/symphony-mcp']
spec=importlib.util.spec_from_file_location('fixture_source','/work/source/tests/test_mcp.py')
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
fake=root/'qxctl-fixture'
fake.write_text(module.FAKE_SOURCE.replace('PYTHON_EXECUTABLE',sys.executable).replace('RECORDS_LITERAL',repr(module.RECORDS)))
fake.chmod(0o700)
log=root/'calls.jsonl'
manifest=json.loads(subprocess.check_output([str(fake),'commands','manifest','--json'],env={'FIXTURE_LOG':str(log)}))
profile={'schema':'symphony.agent.connection.v1','qxctl':str(fake),'qxctl_sha256':'sha256:'+hashlib.sha256(fake.read_bytes()).hexdigest(),
         'working_directory':str(root),'registry_digest':manifest['registry_digest'],'environment':{'FIXTURE_LOG':str(log)},
         'inherit_environment':[],'timeout_ms':5000,'max_output_bytes':16384,'enabled_nonread_commands':[]}
path=root/'connection.json';path.write_text(json.dumps(profile));path.chmod(0o600)
