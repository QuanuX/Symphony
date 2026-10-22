#!/usr/bin/env python3
"""Run on Windows; exercise the actual wsl.exe adapter without installed Symphony."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

def validate(stdout):
    rows=[json.loads(line) for line in stdout.splitlines() if line.strip()]
    by_id={row.get('id'):row for row in rows}
    assert set(by_id)=={1,2,3}, rows
    assert by_id[1]['result']['protocolVersion']=='2025-11-25', rows
    assert len(by_id[2]['result']['tools'])==6, rows
    result=by_id[3]['result']
    assert not result.get('isError',False), rows
    assert result.get('structuredContent') is not None, rows
    return rows

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--distribution',required=True)
    p.add_argument('--output',required=True,type=Path)
    a=p.parse_args()
    if sys.platform!='win32':p.error('This check must execute from Windows')
    argv=['wsl.exe','--distribution',a.distribution,'--user','symphonybuild','--cd','/work',
          '--exec','/work/result/build/symphony-mcp','--config','/work/wsl-fixture/connection.json']
    requests=[{'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2025-11-25','capabilities':{},'clientInfo':{'name':'windows-wsl-fixture','version':'1'}}},
              {'jsonrpc':'2.0','method':'notifications/initialized'},
              {'jsonrpc':'2.0','id':2,'method':'tools/list','params':{}},
              {'jsonrpc':'2.0','id':3,'method':'tools/call','params':{'name':'symphony_read','arguments':{'command_id':'qxcmd:symphony:status'}}}]
    result=subprocess.run(argv,input=''.join(json.dumps(r)+'\n' for r in requests),capture_output=True,text=True,encoding='utf-8',timeout=90)
    a.output.parent.mkdir(parents=True,exist_ok=True)
    report={'argv':argv,'exit_code':result.returncode,'stdout':result.stdout,'stderr':result.stderr,
            'status':'failed','agent_ui_qualified':False,'native_owner_qualified':False}
    try:
        assert result.returncode==0, result.stderr
        validate(result.stdout)
        report['status']='windows_to_wsl_stdio_fixture_passed'
    finally:a.output.write_text(json.dumps(report,indent=2)+'\n')

if __name__=='__main__':main()
