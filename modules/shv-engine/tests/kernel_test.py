"""Independent process checks for SHV's bounded source and semantic contracts."""
import argparse, copy, hashlib, json, os, stat, subprocess, tempfile, time
from pathlib import Path
p=argparse.ArgumentParser(); selected=p.add_mutually_exclusive_group(required=True); selected.add_argument('--engine'); selected.add_argument('--prefix'); p.add_argument('--version',default='0.3.0-dev'); args=p.parse_args()
def canonical(x):return json.dumps(x,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()
def seal(x,key='digest'):
 x=copy.deepcopy(x);x.pop(key,None);x[key]='sha256:'+hashlib.sha256(canonical(x)).hexdigest();return x
calls=0
negative=0
def call(op,payload,good=True):
 global calls,negative
 calls+=1;negative+=not good
 req={'protocol':'symphony.knowledge.engine-process.v1','request_id':'kernel-check','correlation_id':'kernel-check','operation':op,'target_engine':'symphony-shv','deadline_unix_ms':int(time.time()*1000)+30000,'payload':payload}
 r=subprocess.run([args.engine],input=canonical(req),stdout=subprocess.PIPE,stderr=subprocess.PIPE)
 out=json.loads(r.stdout);assert not r.stderr,r.stderr;assert out==seal(out,'response_digest')
 assert (r.returncode==0)==good,(op,r.returncode,out);assert out['outcome']==('ok' if good else 'error'),out
 result=out.get('result')
 if good:assert result==seal(result,'descriptor_digest' if op=='inspect' else 'digest'),result
 return result
def test_kernel_conformance():
 head='<div id="product-overview"><h1>Example CPU™</h1></div>'
 body='<article id="product-specifications"><dl><dt>Cores</dt><dd>16</dd><dt>Sockets</dt><dd>1P / 2P</dd><dt>Launch</dt><dd>03/15/2021</dd><dt>End</dt><dd>done</dd></dl></article>'
 html=head+body
 spec={'id':'cpu','manufacturer':'Example','model':'Example CPU™','hardware_class':'cpu','source_id':'source','heading_section':'div#product-overview','field_section':'article#product-specifications','fields':[{'predicate':'cores','label':'Cores','next_label':'Sockets','value_type':'integer','qualifier':'documented'},{'predicate':'socket_modes','label':'Sockets','next_label':'Launch','value_type':'tokens','qualifier':'finite_supported_set'},{'predicate':'model_introduction','label':'Launch','next_label':'End','value_type':'date','qualifier':'documented'}]}
 with tempfile.TemporaryDirectory(prefix='shv-kernel-') as tmp:
  root=Path(tmp).resolve()
  def build(raw=html,change=None,good=True):
   data=raw if isinstance(raw,bytes) else raw.encode();(root/'source.html').write_bytes(data)
   payload={'source_root':str(root),'sources':[{'id':'source','path':'source.html','bytes':len(data),'digest':'sha256:'+hashlib.sha256(data).hexdigest(),'format':'html'}],'subjects':[copy.deepcopy(spec)]}
   if change:change(payload)
   return call('catalogue_build',payload,good)
  build(html.encode()+b'\xff',good=False)
  cat=build();sub=cat['subjects'][0]
  assert build(head[:-6]+body+'</div>')['subjects']==cat['subjects']
  assert build(head[:-6]+'<div id="product-specifications">'+body+'</div></div>')['subjects']==cat['subjects']
  assert sub['introduced']=={'from':'2021-03-15','through':'2021-03-15'}
  assert sub['assertions']==[{'predicate':'cores','value':16,'qualifier':'documented','source_id':'source'},{'predicate':'model_introduction','value':'2021-03-15','qualifier':'documented','source_id':'source'},{'predicate':'socket_modes','value':['1P','2P'],'qualifier':'finite_supported_set','source_id':'source'}]
  assert build(head.replace('CPU™','CPU&trade;')+body)['subjects']==cat['subjects']
  assert build('<nav><h1>Wrong</h1><dt>Cores</dt><dd>99</dd></nav>'+html)['subjects']==cat['subjects']
  assert build('<script>"</scriptx><dt>Cores</dt><dd>99</dd>"</script>'+html)['subjects']==cat['subjects']
  assert build(head+body.replace('<dl>','<dl><script>"</scriptx><dt>Cores</dt><dd>99</dd>"</script>'))['subjects']==cat['subjects']
  assert build(head+body.replace('<dl>','<dl><dt-fake>Cores</dt-fake><dd-fake>99</dd-fake>'))['subjects']==cat['subjects']
  for raw in [html.replace('<h1>','<h1-fake>').replace('</h1>','</h1-fake>'),html.replace('<dt>Cores</dt>','<dt-fake>Cores</dt-fake>'),html.replace('Example CPU™','Different CPU'),head+head+body,head+body+body,head+body.replace('product-specifications','unrelated'),head+body.replace('</article>',''),html.replace('03/15/2021','02/29/2021'),html.replace('<dd>16</dd>','<dd>016</dd>'),html.replace('1P / 2P','1P / 1P'),html.replace('<dt>End</dt>','<dt>Cores</dt>'),html.replace('CPU™','CPU&unknown;'),html+'\0']:
   build(raw,good=False)
  for change in [lambda p:p['sources'][0].update(digest='sha256:'+'0'*64),lambda p:p['sources'][0].update(bytes=1),lambda p:p['sources'][0].update(path='../source.html'),lambda p:p['sources'][0].update(format='opaque'),lambda p:p['subjects'][0]['fields'].append(copy.deepcopy(p['subjects'][0]['fields'][0])),lambda p:p['subjects'][0]['fields'][0].update(next_label='End'),lambda p:p['subjects'][0].update(field_section='div#product-overview'),lambda p:p.update(source_root='relative')]:build(change=change,good=False)
  (root/'link.html').symlink_to(root/'source.html');build(change=lambda p:p['sources'][0].update(path='link.html'),good=False)
  cat=build();query={'source_root':str(root),'catalogue':cat,'subject_ids':['missing','cpu']}
  qr=call('catalogue_query',query);assert qr['missing_subject_ids']==['missing'] and qr['subjects']==cat['subjects']
  reqs=[{'id':'enough','predicate':'cores','operator':'gte','value':8,'qualifier':'documented'},{'id':'too_many','predicate':'cores','operator':'gte','value':32,'qualifier':'documented'},{'id':'dual','predicate':'socket_modes','operator':'contains','value':'2P','qualifier':'finite_supported_set'},{'id':'quad','predicate':'socket_modes','operator':'contains','value':'4P','qualifier':'finite_supported_set'},{'id':'board','predicate':'whole_board_compatible','operator':'eq','value':'yes','qualifier':'documented'}]
  evaluation=call('evaluate',dict(query,requirements=reqs));assert [f['status'] for f in evaluation['findings']]==['supported','contradicted','supported','contradicted','unresolved'];assert evaluation['missing_subject_ids']==['missing']
  wrong=copy.deepcopy(reqs);wrong[0]['value']='eight';call('evaluate',dict(query,requirements=wrong),False)
  call('catalogue_query',dict(query,subject_ids=['cpu','cpu']),False)
  graph=call('graph_project',{'source_root':str(root),'catalogue':cat});assert [n['id'] for n in graph['nodes']]==['source:source','subject:cpu'];assert len(graph['edges'])==3
  assert call('graph_validate',{'source_root':str(root),'graph':graph})['valid'] is True
  forged=copy.deepcopy(cat);forged['subjects'][0]['assertions'][0]['value']=99;forged=seal(forged)
  for op,payload in [('catalogue_query',dict(query,catalogue=forged)),('evaluate',dict(query,catalogue=forged,requirements=reqs)),('graph_project',{'source_root':str(root),'catalogue':forged})]:call(op,payload,False)
  badgraph=copy.deepcopy(graph);badgraph['edges'][0]['properties']['value']=99;call('graph_validate',{'source_root':str(root),'graph':seal(badgraph)},False)
  (root/'source.html').write_text(html.replace('<dd>16</dd>','<dd>99</dd>'));call('catalogue_query',query,False);call('graph_validate',{'source_root':str(root),'graph':graph},False)
 profile=call('coverage_default',{'as_of':'2026-09-13'});assert profile['selector']['from']=='2018-01-01'
 def summary(id,date):return {'id':id,'manufacturer':'Example','model':id,'hardware_class':'cpu','introduced':date}
 subjects=[summary('old',{'from':'2017-01-01','through':'2017-01-01'}),summary('new',{'from':'2021-03-15','through':'2021-03-15'}),summary('unknown',None),summary('straddle',{'from':'2017-01-01','through':'2019-01-01'})]
 result=call('coverage_plan',{'profile':profile,'subjects':subjects});assert result['counts']=={'included':1,'excluded':1,'unresolved':2}
 def plan(selector):return call('coverage_plan',{'profile':seal(dict(profile,selector=selector)),'subjects':subjects})
 assert plan({'op':'all'})['counts']=={'included':4,'excluded':0,'unresolved':0}
 assert plan({'op':'date','basis':'model_introduction','from':'2010-01-01','through':'2026-09-13'})['counts']=={'included':3,'excluded':0,'unresolved':1}
 assert plan({'op':'not','arg':profile['selector']})['counts']=={'included':1,'excluded':1,'unresolved':2}
 assert plan({'op':'and','args':[profile['selector'],{'op':'ids','values':[]}]})['counts']['excluded']==4
 assert plan({'op':'or','args':[profile['selector'],{'op':'all'}]})['counts']['included']==4
 call('coverage_default',{'as_of':'2026-02-29'},False)
 call('coverage_plan',{'profile':seal(dict(profile,selector=dict(profile['selector'],basis='manufacture'))),'subjects':subjects},False)
 call('coverage_plan',{'profile':dict(profile,as_of='2026-09-12'),'subjects':subjects},False)
 call('inspect',{'extra':True},False)
 d=call('inspect',{});assert len(d['operations'])==8 and d['network_listener'] is False
 print(f'PASS: {calls} independent native process cases, including {negative} rejection cases; scoped extraction, replay, coverage, requirements and graph ownership')

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
 root=Path(prefix).absolute();module='shv-engine';engine='symphony-shv';version='0.3.0-dev'
 receipt_path=f'share/symphony/receipts/{module}/{version}/install-receipt.json'
 raw,_=receipt_read(root,receipt_path,1048576);receipt=json.loads(raw)
 assert receipt==seal(receipt,'receipt_digest')
 for key,value in {'protocol':'symphony.knowledge.install-receipt.v2','format_version':2,'component_id':module,'module_id':module,'package_id':module,'engine_id':engine,'vector_id':'shv','version':version,'component_kind':'vector_engine'}.items():assert receipt[key]==value,(key,receipt[key])
 relative=f'libexec/symphony/{module}/{version}/{engine}'
 entries=[e for e in receipt['entry_points'] if e['entry_point_id']==engine]
 assert len(entries)==1 and entries[0]['path']==relative and entries[0]['kind']=='executable' and entries[0]['protocols']==['symphony.knowledge.engine-process.v1']
 owned=[f for f in receipt['files'] if f['path']==relative];assert len(owned)==1 and owned[0]['kind']=='executable'
 binary,info=receipt_read(root,relative,4194304);assert info.st_mode & 0o111 and len(binary)==owned[0]['size'] and 'sha256:'+hashlib.sha256(binary).hexdigest()==owned[0]['digest']
 return str(root/relative)

def test_installed_process(prefix):
 args.engine=installed_engine(prefix)
 test_kernel_conformance()
 print('PASS: executed exact receipt-owned installed entry point after no-follow identity/size/digest verification')

if __name__=='__main__':
 if args.prefix:test_installed_process(args.prefix)
 else:test_kernel_conformance()
