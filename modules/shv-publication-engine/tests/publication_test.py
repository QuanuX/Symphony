"""Pure owner conformance; synthetic references do not establish live authorization."""
import argparse, copy, hashlib, json, subprocess, time
from pathlib import Path
ENGINE=None
calls=rejects=0

def seal(v):
 v=copy.deepcopy(v);v.pop('digest',None);v['digest']='sha256:'+hashlib.sha256(json.dumps(v,sort_keys=True,separators=(',',':')).encode()).hexdigest();return v

def call(op,p,fail=False):
 global calls,rejects
 request=dict(protocol='symphony.knowledge.engine-process.v1',request_id='publication-test',correlation_id='publication-test',target_engine='symphony-shv-publication',operation=op,deadline_unix_ms=int(time.time()*1000)+30000,payload=p)
 r=subprocess.run([ENGINE],input=json.dumps(request),capture_output=True,text=True);calls+=1
 out=json.loads(r.stdout);assert bool(r.returncode)==fail,(op,r.stdout,r.stderr);assert out['outcome']==('error' if fail else 'ok');assert not r.stderr
 if fail:rejects+=1;return
 result=out['result']
 if op!='inspect':assert seal(result)==result
 return result

def test_native_publication_contract(engine):
 global ENGINE, calls, rejects
 ENGINE=engine
 calls=rejects=0
 h='sha256:'+'a'*64
 inst=dict(Role='shv-partition-engine',ModuleID='shv-partition-engine',EngineID='symphony-shv-partition',Version='0.2.0-dev',Prefix='/partition',ReceiptPath='/partition/share/symphony/receipts/shv-partition-engine/0.2.0-dev/install-receipt.json',ReceiptDigest=h,ReceiptProtocol='symphony.knowledge.install-receipt.v2',ExecutablePath='/partition/libexec/symphony/shv-partition-engine/0.2.0-dev/symphony-shv-partition',ExecutableDigest=h)
 manifest=seal(dict(protocol='symphony.shv.partition-manifest.v1',entries=[],required_references=[],loaded_count=0,missing_count=0,complete_inventory=True,reference_statuses=[]))
 d=dict(catalogue_id='research',tops_id='00000000-0000-4000-8000-000000000001',manifest=manifest,policy=dict(missing_partitions='reject',missing_references='reject'),partition_installation=inst,members=[])
 base=dict(operation_id='first',current=None,desired=d,reason='caller selection')
 descriptor=call('inspect',{});assert len(descriptor['operations'])==4
 plan=call('publication_plan',base);tr=call('publication_reduce',dict(current=None,plan=plan));head=tr['head'];assert head['generation']==1
 assert call('publication_status',dict(history=[head]))['head']==head
 call('publication_plan',{**base,'current':head},True)
 for k,v in [('catalogue_id','other'),('tops_id','00000000-0000-4000-8000-000000000002')]:call('publication_plan',{**base,'current':head,'desired':{**d,k:v}},True)
 for k,v in [('generation',2),('previous_digest',h),('digest',h)]:
  bad=copy.deepcopy(plan);bad['head'][k]=v
  if k!='digest':bad['head']=seal(bad['head'])
  call('publication_reduce',dict(current=None,plan=seal(bad)),True)
 for k,v in [('members',None),('members',[{}]),('policy',{}),('extra',True)]:call('publication_plan',{**base,'desired':{**d,k:v}},True)
 for k,v in [('Version','0.3.0-dev'),('Prefix','/partition/../bad'),('ReceiptDigest','x')]:call('publication_plan',{**base,'desired':{**d,'partition_installation':{**inst,k:v}}},True)
 missing=seal({**manifest,'entries':[dict(partition_digest=h,partition=None)],'missing_count':1,'complete_inventory':False})
 call('publication_plan',{**base,'desired':{**d,'manifest':missing}},True)
 allowed={**d,'manifest':missing,'policy':dict(missing_partitions='allow',missing_references='allow')}
 assert call('publication_plan',{**base,'desired':allowed})['head']['definition']==allowed
 refs=[dict(partition_digest=h,subject_id='cpu')]
 missing=seal({**missing,'required_references':refs,'reference_statuses':[{'reference':refs[0],'status':'missing_partition'}]})
 call('publication_plan',{**base,'desired':{**allowed,'manifest':missing,'policy':dict(missing_partitions='allow',missing_references='reject')}},True)
 call('publication_plan',{**base,'desired':{**allowed,'manifest':missing}})
 call('publication_status',dict(history=[]),True)
 call('publication_status',dict(history=[head,head]),True)
 history=[head]
 for n in range(2,33):
  desired={**d,'policy':dict(missing_partitions='allow' if n%2==0 else 'reject',missing_references='reject')}
  nxt=call('publication_plan',dict(operation_id='r'+str(n),current=history[-1],desired=desired,reason='caller policy change'))
  call('publication_reduce',dict(current=None,plan=nxt),True)
  history.append(call('publication_reduce',dict(current=history[-1],plan=nxt))['head'])
 assert len(call('publication_status',dict(history=history))['history_digests'])==32
 call('publication_plan',{**base,'current':history[-1]},True)
 call('publication_status',dict(history=history+[history[-1]]),True)
 for version in ['0.2.0-dev','0.3.0-dev','0.4.0-dev','0.5.0-dev','latest']:
  selected=copy.deepcopy(inst)
  for key in ['ReceiptPath','ExecutablePath']:selected[key]=selected[key].replace('0.2.0-dev',version)
  selected['Version']=version
  call('publication_plan',{**base,'desired':{**d,'partition_installation':selected}},not (version=='0.2.0-dev' or (descriptor['engine_version']=='0.4.0-dev' and version in ['0.3.0-dev','0.4.0-dev'])))
 print(json.dumps(dict(status='passed',calls=calls,rejections=rejects)))


if __name__ == "__main__":
 p=argparse.ArgumentParser();p.add_argument("--engine",required=True);a=p.parse_args()
 test_native_publication_contract(a.engine)
