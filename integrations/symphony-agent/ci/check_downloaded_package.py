#!/usr/bin/env python3
"""Validate the selected build's binary binding before isolated package tests."""
import argparse
import hashlib
import json
from pathlib import Path
import platform
import subprocess
import sys
import zipfile

p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--download',required=True,type=Path)
p.add_argument('--ubuntu',required=True,choices=['22.04','24.04','26.04'])
p.add_argument('--arch',required=True,choices=['x86_64','arm64'])
p.add_argument('--docker-arch',required=True,choices=['amd64','arm64'])
a=p.parse_args()
observed={'x86_64':'x86_64','aarch64':'arm64','arm64':'arm64'}.get(platform.machine())
assert observed==a.arch, 'Native runner architecture differs from selected target'
reports=list(a.download.glob('**/result/RESULTS.json'))
assert len(reports)==1, 'Expected exactly one release result'
q=json.loads(reports[0].read_text())
assert q['status']=='bridge_tests_passed_on_observed_host'
assert q['binary_architecture']==a.arch
assert q['facts']['os_release']['VERSION_ID']==a.ubuntu
archives=list(a.download.glob('**/packages/*.zip'))
assert len(archives)==1, 'Expected exactly one platform package'
with zipfile.ZipFile(archives[0]) as z:
    assert hashlib.sha256(z.read('bin/symphony-mcp')).hexdigest()==q['binary_sha256']
subprocess.run([sys.executable,str(Path(__file__).with_name('test_package_container.py')),
                '--ubuntu',a.ubuntu,'--arch',a.docker_arch,'--archive',str(archives[0])],check=True)
