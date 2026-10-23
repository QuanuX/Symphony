#!/usr/bin/env python3
"""Prepare disposable local reviewer inputs/profiles; does not apply a mutation."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--qxctl',required=True,type=Path)
    p.add_argument('--working-directory',required=True,type=Path)
    p.add_argument('--output-directory',required=True,type=Path,help='New private directory; existing paths are refused')
    a=p.parse_args()
    qx=a.qxctl.resolve(strict=True);cwd=a.working_directory.resolve(strict=True)
    if not qx.is_file() or not cwd.is_dir():p.error('Select an executable and existing working directory')
    root=a.output_directory.absolute()
    root.mkdir(mode=0o700,parents=False,exist_ok=False)
    root=root.resolve(strict=True)
    helper=Path(__file__).resolve().with_name('configure_connection.py')
    for filename,enabled in [('read-only.json',[]),('configuration-write.json',['--enable','qxcmd:symphony:sov.target.configure'])]:
        subprocess.run([sys.executable,str(helper),'--qxctl',str(qx),'--working-directory',str(cwd),'--output',str(root/filename)]+enabled,check=True,stdout=subprocess.DEVNULL,timeout=35)
    env={'PATH':'/usr/bin:/bin:/usr/sbin:/sbin','LANG':'C.UTF-8'}
    env.update({k:os.environ[k] for k in ('HOME','TMPDIR') if k in os.environ})
    r=subprocess.run([str(qx),'sov','remote','template','--operation','target_configure','--json'],cwd=cwd,env=env,stdin=subprocess.DEVNULL,capture_output=True,timeout=30,check=True)
    value=json.loads(r.stdout)
    if value.get('protocol')!='symphony.sov.remote-template.v1':raise ValueError('Unexpected template protocol')
    request=value['template']
    if request.get('protocol')!='symphony.sov.target-configure-input.v1':raise ValueError('Unexpected configure protocol')
    request['path']=str(root/'configuration.json')
    fd=os.open(root/'create.json',os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(fd,'w') as f:json.dump(request,f,indent=2);f.write('\n')
    print(json.dumps({'directory':str(root),'input':str(root/'create.json'),'owner_configuration':request['path'],'read_only_profile':str(root/'read-only.json'),'write_profile':str(root/'configuration-write.json'),'configuration_applied':False,'target_job_references':'synthetic native template; must not be invoked'},indent=2))

if __name__=='__main__':
    try:main()
    except (OSError,ValueError,KeyError,subprocess.SubprocessError) as e:
        print('Reviewer setup stopped: '+str(e),file=sys.stderr);sys.exit(2)
