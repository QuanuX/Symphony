"""Focused process acceptance with independent expectations and resealed false artifacts."""
import argparse, copy, hashlib, json, os, stat, subprocess, tempfile, time
from pathlib import Path
P=argparse.ArgumentParser();choice=P.add_mutually_exclusive_group(required=True);choice.add_argument('--engine');choice.add_argument('--prefix');A=P.parse_args(); count=0;rejects=0

def digest(x):return 'sha256:'+hashlib.sha256(json.dumps(x,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()).hexdigest()
def seal(x):x=copy.deepcopy(x);x.pop('digest',None);x['digest']=digest(x);return x

def call(op,p,fail=False):
 global count,rejects
 req={'protocol':'symphony.knowledge.engine-process.v1','request_id':'test','correlation_id':'test','target_engine':'symphony-shv-source','operation':op,'deadline_unix_ms':int(time.time()*1000)+30000,'payload':p}
 r=subprocess.run([A.engine],input=json.dumps(req),capture_output=True,text=True);count+=1
 envelope=json.loads(r.stdout);assert not r.stderr and envelope['outcome']==('error' if fail else 'ok')
 unsigned=copy.deepcopy(envelope);signed=unsigned.pop('response_digest');assert signed==digest(unsigned)
 if fail:
  rejects+=1;assert r.returncode!=0,(op,'false artifact accepted',r.stdout);return
 assert r.returncode==0,(op,r.stderr,r.stdout)
 out=json.loads(r.stdout)['result']
 if op!='inspect':assert out==seal(out)
 return out

def desired():return {'source_id':'cpu-family','publisher':'Original component designer','authority_role':'component-specification','subject_ids':['cpu-a','cpu-b'],'locators':[{'id':'spec','uri':'https://example.com/spec','format':'html'}]}
def proposed(current=None,d=None,op='onboard'):return {'operation_id':op,'current':current,'desired':d or desired(),'reason':'Caller-selected source'}
def test_source_conformance():
 with tempfile.TemporaryDirectory() as tmp:
  root=Path(tmp).resolve();body=b'<h1>Component evidence fixture</h1>'; (root/'source.html').write_bytes(body)
  desc=call('inspect',{});assert len(desc['operations'])==8
  for o in desc['operations']:
   assert o['mutability']=='read_only' and o['authorization_requirement']=='none'
   assert o['expected_state_required']==(o['operation_name'] in ['source_plan','source_reduce'])
  p1=call('source_plan',proposed());s1=p1['source'];assert p1['change_kind']=='onboard' and s1['generation']==1 and s1['previous_digest'] is None
  t1=call('source_reduce',{'current':None,'plan':p1});assert t1['source']==s1
  d=desired();d['locators'][0]['uri']='https://new.example.net/spec'
  p2=call('source_plan',proposed(s1,d,'move'));s2=p2['source'];assert p2['change_kind']=='relocation' and s2['generation']==2 and s2['previous_digest']==s1['digest']
  call('source_reduce',{'current':s1,'plan':p2})
  d2=copy.deepcopy(d);d2['publisher']='Variant manufacturer';p3=call('source_plan',proposed(s2,d2,'authority'));assert p3['change_kind']=='authority_change'
  status=call('source_status',{'history':[s1,s2,p3['source']]});assert status['history_digests']==[s1['digest'],s2['digest'],p3['source']['digest']]
  call('source_plan',proposed(s1,desired()),True)
  bad=desired();bad['source_id']='different';call('source_plan',proposed(s1,bad),True)
  for field,v in [('generation',3),('previous_digest','sha256:'+'0'*64)]:
   forged=copy.deepcopy(p2);forged['source']=seal({**forged['source'],field:v});call('source_reduce',{'current':s1,'plan':seal(forged)},True)
  forged=seal({**p2,'change_kind':'onboard'});call('source_reduce',{'current':s1,'plan':forged},True)
  call('source_reduce',{'current':s2,'plan':p2},True)
  for h in [[],[s2],[s1,p3['source']],[s1,s1],[s2,s1]]:call('source_status',{'history':h},True)
  for field in ['publisher','authority_role','subject_ids']:
   d=desired();d[field]=['cpu-c'] if field=='subject_ids' else 'Different scope'
   assert call('source_plan',proposed(s1,d))['change_kind']=='authority_change'
  for u in ['file:///source','https://user:pass@example.com/spec','https://example.com/#secret','https://:80/path','https://example.com:080/path','https://example.com:65536/path','https://%65xample.com/','https://[::1]/','https://example.com/\\x','https://example.com/"x','HTTPS://example.com/','https://example.com/é']:
   d=desired();d['locators'][0]['uri']=u;call('source_plan',proposed(None,d),True)
  for field in ['subject_ids','locators']:
   d=desired();d[field]=[];call('source_plan',proposed(None,d),True)
   d=desired();d[field].append(copy.deepcopy(d[field][0]));call('source_plan',proposed(None,d),True)
  maxed=seal({**s2,'generation':32});call('source_plan',proposed(maxed,desired()),True)
  p={'source_root':str(root),'source':s1,'locator_id':'spec','resolved_uri':'https://example.com/spec','redirect_chain':[],'observed_at':'2026-09-13T19:00:00Z','upstream_revision':None,'manifest':{'id':'capture-a','path':'source.html','bytes':len(body),'digest':'sha256:'+hashlib.sha256(body).hexdigest(),'format':'html'},'completeness':'complete','issues':[]}
  c1=call('capture_import',p);assert c1['source']==s1 and 'source_root' not in c1
  now=copy.deepcopy(p);now['observed_at']='2026-09-13T19:00:01Z';c2=call('capture_import',now)
  cmp=call('capture_compare',{'source_root':str(root),'previous':c1,'current':c2});assert cmp['changes']==['observation_time'] and cmp['same_logical_source']
  moved=copy.deepcopy(p);moved['source']=s2;moved['resolved_uri']='https://new.example.net/spec';c3=call('capture_import',moved)
  cmp=call('capture_compare',{'source_root':str(root),'previous':c1,'current':c3});assert cmp['changes']==['source_revision','acquisition_route']
  for field,v in [('locator_id','missing'),('resolved_uri','https://elsewhere.example/spec'),('redirect_chain',['https://example.com/spec']),('observed_at','2026-02-30T19:00:00Z'),('observed_at','2026-09-13T19:00:60Z'),('completeness','partial')]:
   call('capture_import',{**p,field:v},True)
  partial={**p,'completeness':'partial','issues':['Only a subsection retained']};cp=call('capture_import',partial);assert cp['completeness']=='partial'
  redirect={**p,'resolved_uri':'https://new.example.org/spec','redirect_chain':['https://new.example.org/spec']};cr=call('capture_import',redirect);assert cr['source']==s1
  for field,v in [('bytes',len(body)+1),('digest','sha256:'+'0'*64),('path','../source.html'),('format','opaque')]:
   call('capture_import',{**p,'manifest':{**p['manifest'],field:v}},True)
  (root/'link.html').symlink_to(root/'source.html');call('capture_import',{**p,'manifest':{**p['manifest'],'path':'link.html'}},True)
  newer=copy.deepcopy(now);newer['manifest']['id']='capture-b';c4=call('capture_import',newer)
  g=call('graph_project',{'source_root':str(root),'captures':[c4,c1]});assert len(g['nodes'])==3 and len(g['edges'])==2
  assert [n['id'] for n in g['nodes']]==sorted(n['id'] for n in g['nodes'])
  call('graph_validate',{'source_root':str(root),'graph':g})
  forged=copy.deepcopy(g);forged['edges'][0]['properties']['source_id']='wrong';call('graph_validate',{'source_root':str(root),'graph':seal(forged)},True)
  call('graph_project',{'source_root':str(root),'captures':[c1,c1]},True)
  call('graph_project',{'source_root':str(root),'captures':[c1]*9},True)
  call('graph_project',{'source_root':str(root)+'/', 'captures':[]},True)
  call('capture_import',{**p,'source_root':str(root)+'/'},True)
  call('graph_project',{'source_root':str(root)+'\x01', 'captures':[]},True)
  empty=call('graph_project',{'source_root':str(root),'captures':[]});assert empty['nodes']==[];call('graph_validate',{'source_root':str(root),'graph':empty})
  (root/'source.html').write_bytes(b'changed');call('capture_compare',{'source_root':str(root),'previous':c1,'current':c2},True);call('graph_validate',{'source_root':str(root),'graph':g},True)
  call('source_plan',{**proposed(),'approved':True},True)
def receipt_read(root, relative, maximum):
 parts=Path(root).parts
 assert Path(root).is_absolute() and '..' not in parts
 fd=os.open('/',os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW)
 try:
  for part in list(parts[1:])+list(Path(relative).parts[:-1]):
   assert part not in ('','.','..')
   nxt=os.open(part,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW,dir_fd=fd);os.close(fd);fd=nxt
  leaf=os.open(Path(relative).name,os.O_RDONLY|os.O_NOFOLLOW,dir_fd=fd)
  try:
   info=os.fstat(leaf);assert stat.S_ISREG(info.st_mode) and info.st_size<=maximum
   data=b''
   while True:
    block=os.read(leaf,65536)
    if not block:break
    data+=block;assert len(data)<=maximum
   return data,info
  finally:os.close(leaf)
 finally:os.close(fd)

def installed_engine(prefix):
 root=Path(prefix).absolute();module='shv-source-engine';engine='symphony-shv-source';version='0.1.0-dev'
 receipt_path=f'share/symphony/receipts/{module}/{version}/install-receipt.json'
 raw,_=receipt_read(root,receipt_path,1048576);receipt=json.loads(raw)
 unsigned=copy.deepcopy(receipt);signed=unsigned.pop('receipt_digest');assert signed==digest(unsigned)
 for key,value in {'protocol':'symphony.knowledge.install-receipt.v2','format_version':2,'component_id':module,'module_id':module,'package_id':module,'engine_id':engine,'vector_id':'shv','version':version,'component_kind':'vector_engine'}.items():assert receipt[key]==value,(key,receipt[key])
 relative=f'libexec/symphony/{module}/{version}/{engine}'
 entries=[e for e in receipt['entry_points'] if e['entry_point_id']==engine]
 assert len(entries)==1 and entries[0]['path']==relative and entries[0]['kind']=='executable' and entries[0]['protocols']==['symphony.knowledge.engine-process.v1']
 owned=[f for f in receipt['files'] if f['path']==relative];assert len(owned)==1 and owned[0]['kind']=='executable'
 binary,info=receipt_read(root,relative,4194304);assert info.st_mode & 0o111 and len(binary)==owned[0]['size'] and 'sha256:'+hashlib.sha256(binary).hexdigest()==owned[0]['digest']
 return str(root/relative)

def test_installed_process(prefix):
 A.engine=installed_engine(prefix)
 test_source_conformance()
 print('PASS: executed exact receipt-owned installed entry point after no-follow identity/size/digest verification')


if __name__=='__main__':
 if A.prefix:test_installed_process(A.prefix)
 else:test_source_conformance()
 print(json.dumps({'status':'passed','process_cases':count,'rejection_cases':rejects,'scope':'new SHV source lifecycle and graph replay only'}))
