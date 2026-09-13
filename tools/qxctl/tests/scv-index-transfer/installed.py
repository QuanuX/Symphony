#!/usr/bin/env python3
"""Focused real C++ owners plus production/test-only qxctl transfer campaign."""
import argparse,json,pathlib,sys,os,signal,subprocess,time,hashlib
HERE=pathlib.Path(__file__).resolve();sys.path.insert(0,str(HERE.parents[4]/'modules/scv-graph-duckdb-connector/tests'))
from installed_integration import Campaign,canonical,digest
class TransferCampaign(Campaign):
 def plan(self,name,ids,target=None,legacy=False):
  inv=self.qx(name+'-inventory','inventory',{'expected_revision':None,'cursor':None,'limit':16})['connector_result']
  root=target or self.out/name;root.mkdir(mode=0o700)
  p=self.qx(name+'-plan','transfer-plan',{'expected_revision':inv['manifest']['digest'],'operation_ids':ids,'target':{'prefix':str((self.args.legacy_prefix if legacy else self.args.connector_prefix).resolve()),'version':'0.1.0-dev' if legacy else '0.2.0-dev','root':str(root)},'capacity':{'intents':128,'snapshots':128}})['connector_result']
  return {'plan':p,'expected_plan_digest':p['digest']}
 def run(self):
  self.summary['recovery_scope']='Test-only coordinator SIGKILL at five durable/native boundaries; exact installed production recovery.'
  first=self.qx('source-import','import',{'operation_id':'original','graph':self.graph,'query_time':self.time})['connector_result'];snap=first['intent']['snapshot']
  alias={'tops_id':self.tops,'namespace':self.namespace,'operation_id':'prepared-alias','graph':self.graph,'query_time':self.time,'owner':snap['owner'],'connector':snap['connector']}
  self.native('source-prepare-alias','prepare',alias,snap['connector'])
  request=self.plan('normal',['original','prepared-alias']);before=request['plan']['manifest']
  result=self.qx('execute','transfer',request)
  self.check(result['status']=='complete','Transfer completes exact caller selection')
  manifest=result['target_manifest'];self.check(len(manifest['entries'])==2 and len(manifest['snapshots'])==1,'Target shared snapshot and both operation identities retained')
  self.check({e['operation_id']:e['state'] for e in manifest['entries']}=={'original':'committed','prepared-alias':'prepared'},'Prepared source remains prepared')
  replay=self.qx('replay','transfer',request);self.check(replay==result,'Completed exact replay revalidates without new progress')
  root=request['plan']['input']['target_root']
  status=self.execute('status',[str(self.args.qxctl.resolve()),'scv','graph-index','transfer-status','--target-root',root,'--transfer-digest',result['transfer_digest'],'--json'])
  self.check(status['status']=='recorded_complete' and status['target_manifest'] is None,'Journal-only inspection distinguishes recorded completion from current verification')
  wrong=dict(request,expected_plan_digest='sha256:'+'0'*64);self.qx('wrong-digest','transfer',wrong,ok=False)
  source=self.qx('source-after','inventory',{'expected_revision':before['digest'],'cursor':None,'limit':16})['connector_result']['manifest'];self.check(source==before,'Source complete manifest unchanged')
  (pathlib.Path(root)/'foreign').write_text('caller data')
  rejected=self.qx('unexpected-target','transfer-recover',request);self.check(rejected['status']=='incomplete' and 'unexpected' in rejected['problem'],'Unexpected target content prevents replay completion')
  (pathlib.Path(root)/'foreign').unlink()
  self.qx('foreign-operation','import',{'operation_id':'foreign-operation','graph':self.graph,'query_time':self.time},root=pathlib.Path(root))
  conflict=self.qx('foreign-operation-reject','transfer-recover',request);self.check(conflict['status']=='incomplete' and 'unexpected' in conflict['problem'],'Unselected native destination operation prevents completion')
  if self.args.fault_qxctl:
   for phase in ['prepare_pending','after_prepare','commit_pending','after_commit','complete']:
    req=self.plan('fault-'+phase,['original']);path=self.out/('fault-'+phase+'-input.json');path.write_bytes(canonical(req))
    command=[str(self.args.fault_qxctl.resolve()),'scv','graph-index','transfer','--connector-prefix',str(self.args.connector_prefix.resolve()),'--connector-version','0.2.0-dev','--index-root',str(self.root),'--tops-id',self.tops,'--namespace',self.namespace,'--input',str(path),'--json']
    env=dict(os.environ,SYMPHONY_TRANSFER_TEST_STOP=phase);proc=subprocess.Popen(command,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    stopped=False;deadline=time.monotonic()+45
    while time.monotonic()<deadline:
     pid,state=os.waitpid(proc.pid,os.WNOHANG|os.WUNTRACED)
     if pid:
      if os.WIFSTOPPED(state):stopped=True;break
      raise AssertionError(('fault process exited before barrier',phase,state,proc.stdout.read(),proc.stderr.read()))
     time.sleep(.05)
    if not stopped:proc.kill();proc.communicate();raise AssertionError(('barrier timeout',phase))
    proc.kill();stdout,stderr=proc.communicate(timeout=5);self.check(proc.returncode==-signal.SIGKILL and stdout==b'','Test-only coordinator killed after named boundary: '+phase)
    self.faults.append({'phase':phase,'observed_stop':True,'exit_code':proc.returncode,'stdout_bytes':len(stdout),'test_cli_sha256':hashlib.sha256(self.args.fault_qxctl.read_bytes()).hexdigest()})
    (self.out/'FAULTS.json').write_text(json.dumps(self.faults,indent=2)+'\n')
    recovered=self.qx('recover-'+phase,'transfer-recover',req);self.check(recovered['status']=='complete','Production qxctl recovers exact transfer after '+phase)
    self.check(len(recovered['target_manifest']['entries'])==1,'Recovery retains one exact operation after '+phase)
  stale=self.plan('stale',['original']);self.qx('source-addition','import',{'operation_id':'later','graph':self.graph,'query_time':self.time})
  self.qx('stale-reject','transfer',stale,ok=False);self.check(not list(pathlib.Path(stale['plan']['input']['target_root']).iterdir()),'Stale source rejected before destination journal or index mutation')
  owner=self.clone_owner();self.qx('copied-owner-import','import',{'operation_id':'copied-owner','graph':self.graph,'query_time':self.time},owner=owner)
  missing=self.plan('missing-owner',['copied-owner']);executable=pathlib.Path(missing['plan']['selected'][0]['source']['intent']['snapshot']['owner']['ExecutablePath']);hidden=executable.with_suffix('.hidden');executable.rename(hidden)
  try:
   self.qx('missing-owner-reject','transfer',missing,ok=False);self.check(not list(pathlib.Path(missing['plan']['input']['target_root']).iterdir()),'Missing original semantic owner prevents reservation')
  finally: hidden.rename(executable)
  legacy=self.plan('legacy-target',['original','prepared-alias'],legacy=True);legacy_result=self.qx('legacy-target-execute','transfer',legacy);self.check(legacy_result['status']=='complete' and all(e['connector']['Version']=='0.1.0-dev' for e in legacy_result['target_manifest']['entries']),'Exact legacy writer target completes with preserved version identities')
  self.summary.update(status='passed',fault_cases=len(self.faults),source_records_deleted=0,scope='Recoverable target copy only; no retirement, graph-head switch or database migration');self.save()
if __name__=='__main__':
 p=argparse.ArgumentParser()
 for n in ['qxctl','connector-prefix','owner-prefix','graph','out','legacy-prefix']:p.add_argument('--'+n,type=pathlib.Path,required=True)
 p.add_argument('--fault-qxctl',type=pathlib.Path);a=p.parse_args();a.connector_version='0.2.0-dev';c=TransferCampaign(a);c.faults=[]
 try:c.run()
 except Exception:c.summary['status']='failed';c.save();raise
 print(json.dumps({'status':'passed','calls':len(c.calls),'assertions':len(c.assertions),'faults':len(c.faults)}))
