#!/usr/bin/env python3
"""Test a packaged bridge in a fresh matching Ubuntu with no compiler installed."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import tempfile
import uuid
import zipfile

ROOT=Path(__file__).resolve().parents[1]
p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--ubuntu',required=True,choices=['22.04','24.04','26.04'])
p.add_argument('--arch',required=True,choices=['amd64','arm64'])
p.add_argument('--archive',required=True,type=Path)
a=p.parse_args()
out=ROOT/'evidence'/f'package-ubuntu-{a.ubuntu}-{a.arch}';out.mkdir(exist_ok=False)
name='symphony-package-'+uuid.uuid4().hex[:10]
host_arch={'x86_64':'amd64','aarch64':'arm64','arm64':'arm64'}.get(platform.machine())
report={'status':'running','archive':a.archive.name,'sha256':hashlib.sha256(a.archive.read_bytes()).hexdigest(),
        'ubuntu':a.ubuntu,'arch':a.arch,'execution':'native_cpu_container' if a.arch==host_arch else 'emulated',
        'compiler_installed':False,'container':name,'commands':[]}
def save(): (out/'RESULTS.json').write_text(json.dumps(report,indent=2)+'\n')
with tempfile.TemporaryDirectory(prefix='symphony-package-') as scratch:
    tmp=Path(scratch);(tmp/'docker').mkdir();(tmp/'docker/config.json').write_text('{"auths":{}}')
    env=dict(os.environ,DOCKER_CONFIG=str(tmp/'docker'))
    sock=Path.home()/'.docker/run/docker.sock'
    if platform.system()=='Darwin' and sock.exists():env['DOCKER_HOST']='unix://'+str(sock)
    package=tmp/'package';package.mkdir()
    with zipfile.ZipFile(a.archive) as z:
        for item in z.infolist():
            rel=Path(item.filename)
            if rel.is_absolute() or '..' in rel.parts or item.is_dir():raise RuntimeError('Unexpected archive entry')
            dst=package/rel;dst.parent.mkdir(parents=True,exist_ok=True);dst.write_bytes(z.read(item))
            dst.chmod(0o755 if str(rel)=='bin/symphony-mcp' else 0o644)
    def run(label,argv,check=True,timeout=1200):
        print(label,flush=True);report['commands'].append({'label':label,'argv':argv});save()
        with (out/(label+'.log')).open('wb') as log:r=subprocess.run(argv,env=env,stdout=log,stderr=subprocess.STDOUT,timeout=timeout)
        report['commands'][-1]['exit_code']=r.returncode;save()
        if check and r.returncode:raise RuntimeError(label+' failed')
    created=False
    try:
        run('create',['docker','create','--platform','linux/'+a.arch,'--name',name,'ubuntu:'+a.ubuntu,'sleep','infinity']);created=True
        run('start',['docker','start',name])
        ssl='libssl3' if a.ubuntu=='22.04' else 'libssl3t64'
        run('provision',['docker','exec','--env','DEBIAN_FRONTEND=noninteractive',name,'sh','-c',
                         'apt-get update && apt-get install -y --no-install-recommends python3 '+ssl+' libstdc++6 && useradd --create-home symphonytest && mkdir /package'])
        run('copy',['docker','cp',str(package)+'/.',name+':/package'])
        run('isolate',['docker','network','disconnect','bridge',name])
        run('inventory',['docker','exec',name,'sh','-c','cat /etc/os-release; uname -a; dpkg-query -W; if command -v gcc || command -v clang; then exit 1; fi'])
        run('version',['docker','exec','--user','symphonytest',name,'/package/bin/symphony-mcp','--version'])
        run('linkage',['docker','exec',name,'ldd','/package/bin/symphony-mcp'])
        run('mcp-fixtures',['docker','exec','--user','symphonytest',name,'python3','/package/tests/test_mcp.py','/package/bin/symphony-mcp'],timeout=300)
        details=(out/'mcp-fixtures.log').read_text()
        if 'Ran 20 tests' not in details or re.search(r'\bskipped\b',details,re.I):raise RuntimeError('Missing or skipped MCP boundary')
        report['status']='packaged_mcp_and_ipc_fixtures_passed'
    except Exception as error:report['status']='failed';report['error']=str(error);raise
    finally:
        if created:run('remove',['docker','rm','-f',name],check=False)
        save()
