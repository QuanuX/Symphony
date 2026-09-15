import argparse,copy,hashlib,json,subprocess,time,os,stat
from pathlib import Path
p=argparse.ArgumentParser();g=p.add_mutually_exclusive_group(required=True);g.add_argument('--engine');g.add_argument('--prefix');a=p.parse_args();engine=a.engine or str(Path(a.prefix)/'libexec/symphony/shv-partition-engine/0.2.0-dev/symphony-shv-partition');count=0;rejects=0
H=lambda n:'sha256:'+hashlib.sha256(str(n).encode()).hexdigest()
def seal(v):v=copy.deepcopy(v);v.pop('digest',None);v['digest']='sha256:'+hashlib.sha256(json.dumps(v,sort_keys=True,separators=(',',':')).encode()).hexdigest();return v
def call(op,p,fail=False):
 global count,rejects
 req={'protocol':'symphony.knowledge.engine-process.v1','request_id':'test','correlation_id':'test','target_engine':'symphony-shv-partition','operation':op,'deadline_unix_ms':int(time.time()*1000)+30000,'payload':p};r=subprocess.run([engine],input=json.dumps(req),capture_output=True,text=True);count+=1;out=json.loads(r.stdout);assert bool(r.returncode)==fail,(op,r.stdout,r.stderr);assert out['outcome']==('error' if fail else 'ok');assert not r.stderr
 if fail:rejects+=1;return
 result=out['result']
 if op!='inspect':assert seal(result)==result
 return result
def fixture():return {'dependencies':{'source_revision_digest':H(1),'source_engine':{'engine_id':'symphony-shv-source','version':'0.1.0-dev','executable_digest':H(2)},'kernel_engine':{'engine_id':'symphony-shv','version':'0.2.0-dev','executable_digest':H(3)},'captures':[{'capture_id':'page','capture_digest':H(4),'content_digest':H(5),'bytes':7}],'mapping_digest':H(6),'catalogue_digest':H(7)},'subject_ids':['cpu-b','cpu-a']}
def test_native_partition_contract():
 desc=call('inspect',{});assert len(desc['operations'])==4
 for sv in ['0.1.0-dev','0.2.0-dev','0.3.0-dev','latest']:
  for kv in ['0.1.0-dev','0.2.0-dev','0.3.0-dev','0.4.0-dev','0.5.0-dev']:
   sample=fixture();sample['dependencies']['source_engine']['version']=sv;sample['dependencies']['kernel_engine']['version']=kv
   accepted=(sv=='0.1.0-dev' or (desc['engine_version']=='0.4.0-dev' and sv=='0.2.0-dev')) and kv!='0.5.0-dev' and (kv!='0.4.0-dev' or desc['engine_version']=='0.4.0-dev')
   call('partition_build',sample,not accepted)
 f=fixture();part=call('partition_build',f);assert part['subject_ids']==['cpu-a','cpu-b'];f['subject_ids'].reverse();assert call('partition_build',f)==part
 refs=[{'partition_digest':part['digest'],'subject_id':'cpu-a'},{'partition_digest':H(8),'subject_id':'cpu-c'},{'partition_digest':part['digest'],'subject_id':'cpu-x'},{'partition_digest':H(9),'subject_id':'cpu-d'}]
 mi={'entries':[{'partition_digest':part['digest'],'partition':part},{'partition_digest':H(8),'partition':None}],'required_references':refs};m=call('manifest_build',mi);assert m['missing_count']==1 and not m['complete_inventory'];assert [s['status'] for s in m['reference_statuses']]==['found','missing_partition','missing_subject','unlisted_partition']
 qi={'manifest':m,'selection':refs,'limit':1,'cursor':None};statuses=[]
 for i in range(4):
  q=call('manifest_query',qi);assert q['offset']==i;statuses+=q['rows'];qi['cursor']=q['next_cursor']
 assert qi['cursor'] is None and statuses==m['reference_statuses']
 qi={'manifest':m,'selection':refs,'limit':1,'cursor':None};q=call('manifest_query',qi)
 for field,value in [('manifest_digest',H(0)),('selection_digest',H(0)),('offset',0),('offset',4),('offset',True),('offset',1.5)]:
  bad=copy.deepcopy(qi);bad['cursor']={**q['next_cursor'],field:value};call('manifest_query',bad,True)
 for field,value in [('complete_inventory',True),('missing_count',0),('reference_statuses',[])]:
  bad=copy.deepcopy(qi);bad['manifest']=seal({**m,field:value});call('manifest_query',bad,True)
 for field,value in [('subject_ids',['x','x']),('subject_ids',[None]),('subject_ids',None),('extra',True)]:call('partition_build',{**f,field:value},True)
 for field,value in [('bytes',True),('bytes',-1),('bytes',1048577),('bytes',1.5),('capture_digest','invalid')]:
  bad=copy.deepcopy(f);bad['dependencies']['captures'][0][field]=value;call('partition_build',bad,True)
 bad=copy.deepcopy(f);bad['dependencies']['captures']*=2;call('partition_build',bad,True)
 for field,value in [('source_engine',{'engine_id':'other','version':'0.1.0-dev','executable_digest':H(2)}),('captures',[])]:
  bad=copy.deepcopy(f);bad['dependencies'][field]=value;call('partition_build',bad,True)
 for entries in [mi['entries']*2,[{'partition_digest':H(0),'partition':part}]]:call('manifest_build',{**mi,'entries':entries},True)
 call('manifest_build',{**mi,'required_references':refs*2},True)
 empty=call('manifest_build',{'entries':[],'required_references':[]});out=call('manifest_query',{'manifest':empty,'selection':[],'limit':32,'cursor':None});assert out['rows']==[] and out['next_cursor'] is None
 call('manifest_query',{**qi,'limit':33},True);call('manifest_query',{**qi,'limit':0},True)
 # Same subject IDs in distinct partitions remain separately addressable.
 second=call('partition_build',{**f,'dependencies':{**f['dependencies'],'mapping_digest':H(22)}});both=call('manifest_build',{'entries':[{'partition_digest':p['digest'],'partition':p} for p in [part,second]],'required_references':[{'partition_digest':p['digest'],'subject_id':'cpu-a'} for p in [part,second]]});assert all(x['status']=='found' for x in both['reference_statuses'])
# Boundaries are exercised by the conformance function through this helper.
def test_inventory_bounds():
 f=fixture();caps=[]
 for i in range(5):caps.append({'capture_id':'page-'+str(i),'capture_digest':H(100+i),'content_digest':H(200+i),'bytes':1048576})
 f['dependencies']['captures']=caps;call('partition_build',f,True);f['dependencies']['captures']=caps[:4];base=call('partition_build',f)
 entries=[]
 for i in range(64):
  value=copy.deepcopy(base);value['dependencies']['mapping_digest']=H(300+i);value=seal(value);entries.append({'partition_digest':value['digest'],'partition':value})
 refs=[{'partition_digest':entry['partition_digest'],'subject_id':id} for entry in entries for id in ['cpu-a','cpu-b']]
 mi={'entries':entries,'required_references':refs};m=call('manifest_build',mi);assert m['loaded_count']==64 and len(m['reference_statuses'])==128
 call('manifest_build',{'entries':entries+[{'partition_digest':H(999),'partition':None}],'required_references':[]},True)
 call('manifest_build',{'entries':entries,'required_references':refs+[{'partition_digest':H(999),'subject_id':'cpu'}]},True)
 q=call('manifest_query',{'manifest':m,'selection':refs,'limit':32,'cursor':None});assert len(q['rows'])==32 and q['next_cursor']['offset']==32

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
 root=Path(prefix).absolute();module='shv-partition-engine';engine='symphony-shv-partition';version='0.2.0-dev'
 receipt_path=f'share/symphony/receipts/{module}/{version}/install-receipt.json'
 raw,_=receipt_read(root,receipt_path,1048576);receipt=json.loads(raw)
 unsigned=copy.deepcopy(receipt);signed=unsigned.pop('receipt_digest');assert signed=='sha256:'+hashlib.sha256(json.dumps(unsigned,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()).hexdigest()
 for key,value in {'protocol':'symphony.knowledge.install-receipt.v2','format_version':2,'component_id':module,'module_id':module,'package_id':module,'engine_id':engine,'vector_id':'shv','version':version,'component_kind':'vector_engine'}.items():assert receipt[key]==value,(key,receipt[key])
 relative=f'libexec/symphony/{module}/{version}/{engine}'
 entries=[e for e in receipt['entry_points'] if e['entry_point_id']==engine]
 assert len(entries)==1 and entries[0]['path']==relative and entries[0]['kind']=='executable' and entries[0]['protocols']==['symphony.knowledge.engine-process.v1']
 owned=[f for f in receipt['files'] if f['path']==relative];assert len(owned)==1 and owned[0]['kind']=='executable'
 binary,info=receipt_read(root,relative,4194304);assert info.st_mode & 0o111 and len(binary)==owned[0]['size'] and 'sha256:'+hashlib.sha256(binary).hexdigest()==owned[0]['digest']
 return str(root/relative)


def test_installed_process():
 global engine
 engine=installed_engine(a.prefix)
 test_native_partition_contract()
if a.prefix:test_installed_process()
else:test_native_partition_contract()
test_inventory_bounds()
print(json.dumps({'status':'passed','calls':count,'rejections':rejects}))
