"""Behavioral conformance for caller vocabularies, portable selection and exact replay.
Synthetic original-component fixtures, not vendor facts or a distributed catalogue.
"""
import argparse,copy,hashlib,json,subprocess,tempfile,time
from pathlib import Path

def canonical(x):return json.dumps(x,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()
def seal(x,key='digest'):
 x=copy.deepcopy(x);x.pop(key,None);x[key]='sha256:'+hashlib.sha256(canonical(x)).hexdigest();return x

def run(engine,root,qxctl=None,prefix=None,cases=None):
 calls=[]
 def call(op,p,good=True):
  req={'protocol':'symphony.knowledge.engine-process.v1','request_id':'profile-check','correlation_id':'profile-check','operation':op,'target_engine':'symphony-shv-profile','deadline_unix_ms':int(time.time()*1000)+30000,'payload':p}
  if qxctl:
   family,leaf=('profile','inspect')if op=='inspect'else op.split('_');input_path=root/'request.json';input_path.write_bytes(canonical(p));cmd=[qxctl,'shv',family,leaf,'--prefix',prefix,'--version','0.1.0-dev','--json'];cmd+=[]if op=='inspect'else ['--input',str(input_path)]
   x=subprocess.run(cmd,capture_output=True)
  else:x=subprocess.run([engine],input=canonical(req),capture_output=True)
  assert (x.returncode==0)==good,(op,good,x.stdout,x.stderr)
  out=json.loads(x.stdout)
  if not qxctl:assert out==seal(out,'response_digest')
  result=out.get('result')
  if good:assert result==seal(result,'descriptor_digest'if op=='inspect'else'digest')
  case={'operation':op,'input':p,'good':good,'result':result,'exit_code':x.returncode};calls.append(case)
  if cases:(cases/f'{len(calls):03d}.json').write_bytes(canonical(case))
  return result
 descriptor=call('inspect',{});assert len(descriptor['operations'])==5
 sources=[];mapping=[];profiles=[]
 metrics=[('gpu','device_memory_bytes','bytes',16),('memory','module_capacity_bytes','bytes',32),('storage','namespace_capacity_bytes','bytes',64),('networking-card','port_line_rate_bps','bits_per_second',128),('custom-accelerator','caller_metric','caller_unit',256)]
 source_root=root/'sources';source_root.mkdir()
 for cls,pred,unit,value in metrics:
  model='Synthetic '+cls
  html=f'<div id="overview"><h1>{model}</h1></div><article id="spec"><dl><dt>Value</dt><dd>{value}</dd><dt>Launch</dt><dd>03/15/2021</dd><dt>End</dt><dd>done</dd></dl></article>'.encode();(source_root/f'{cls}.html').write_bytes(html)
  sources.append({'id':cls,'path':f'{cls}.html','bytes':len(html),'digest':'sha256:'+hashlib.sha256(html).hexdigest(),'format':'html'})
  mapping.append({'id':cls,'manufacturer':'Synthetic fixture','model':model,'hardware_class':cls,'source_id':cls,'heading_section':'div#overview','field_section':'article#spec','fields':[{'predicate':pred,'value_type':'integer','qualifier':'unit='+unit,'label':'Value','next_label':'Launch'},{'predicate':'model_introduction','value_type':'date','qualifier':'documented','label':'Launch','next_label':'End'}]})
  d={'id':cls+'-comparison','revision':'v1','hardware_class':cls,'metrics':[{'predicate':pred,'value_type':'integer','qualifier':'unit='+unit,'required':True,'description':'Synthetic conformance metric; caller defines its meaning.','extensions':{}},{'predicate':'optional_fixture','value_type':'string','qualifier':'caller','required':False,'description':'Missing optional declaration is permitted.','extensions':{}}],'extensions':{'fixture_only':True}}
  pr=call('profile_compile',d);profiles.append(pr)
  result=call('mapping_diagnose',{'profile':pr,'mapping':mapping})
  assert result['counts']=={'conformant':1,'incomplete':0,'not_applicable':len(mapping)-1}
  assert result['subjects'][-1]['extension_predicates']==['model_introduction']
  assert [f['status']for f in result['subjects'][-1]['findings']]==['matched','unmapped_optional']
 diag=call('mapping_diagnose',{'profile':profiles[0],'mapping':mapping});assert diag['counts']['not_applicable']==4
 d=copy.deepcopy(profiles[0]['definition']);d['revision']='v2';d['metrics'][0]['value_type']='string';d['metrics'][0]['qualifier']='unit=other';changed=call('profile_compile',d)
 mismatch=call('mapping_diagnose',{'profile':changed,'mapping':mapping});assert mismatch['subjects'][0]['status']=='incomplete' and mismatch['subjects'][0]['findings'][0]['differences']==['value_type','qualifier']
 d['revision']='v3';d['metrics'][0]['predicate']='future_metric';future=call('profile_compile',d);assert call('mapping_diagnose',{'profile':future,'mapping':mapping})['subjects'][0]['findings'][0]['status']=='unmapped_required'
 d['revision']='v4';d['metrics']=[];retired=call('profile_compile',d);assert call('mapping_diagnose',{'profile':retired,'mapping':mapping})['subjects'][0]['status']=='conformant'
 assert call('mapping_diagnose',{'profile':profiles[0],'mapping':mapping})==diag
 for typ in ['string','integer','date','tokens','quarter_20yy','table_rows']:
  d=copy.deepcopy(profiles[0]['definition']);d['metrics'][0]['value_type']=typ;call('profile_compile',d)
 for mutate in [lambda d:d.update(extra=True),lambda d:d['metrics'].append(d['metrics'][0]),lambda d:d['metrics'][0].update(value_type='float'),lambda d:d['metrics'][0].update(required='yes'),lambda d:d.update(hardware_class=''),lambda d:d.update(extensions=[]),lambda d:d['metrics'][0].update(description='bad\ncontrol')]:
  d=copy.deepcopy(profiles[0]['definition']);mutate(d);call('profile_compile',d,False)
 p=copy.deepcopy(profiles[0]);p['definition']['revision']='tampered';call('mapping_diagnose',{'profile':p,'mapping':mapping},False)
 for mutate in [lambda m:m[0].update(extra=True),lambda m:m.append(m[0]),lambda m:m[0]['fields'].append(m[0]['fields'][0]),lambda m:m[0]['fields'][0].update(value_type='float')]:
  m=copy.deepcopy(mapping);mutate(m);call('mapping_diagnose',{'profile':profiles[0],'mapping':m},False)
 coverage=seal({'protocol':'symphony.shv.coverage-profile.v1','as_of':'2026-09-14','selector':{'op':'date','basis':'model_introduction','from':'2018-01-01','through':'2026-09-14'}})
 definition={'id':'caller-universe','revision':'v1','kernel_version':'0.3.0-dev','coverage':coverage,'profiles':profiles[:4],'sources':sources,'mapping':mapping,'locators':[{'source_id':s['id'],'uri':'local-reference:'+s['path'],'upstream_revision':None}for s in sources],'extensions':{'fixture_only':True}}
 u=call('universe_build',definition);bindings={'source_root':str(source_root),'decoders':{}}
 bound=call('universe_bind',{'universe':u,'bindings':bindings});assert bound['coverage']['counts']=={'included':5,'excluded':0,'unresolved':0};assert bound['unprofiled_classes']==['custom-accelerator'];assert bound['canonical_apply_enabled']==False
 assert [s['assertions'][0]['value']for s in bound['catalogue']['subjects'] if s['id']=='gpu']==[16]
 alternate=root/'alternate';alternate.mkdir()
 for s in sources:(alternate/s['path']).write_bytes((source_root/s['path']).read_bytes())
 relocated=call('universe_bind',{'universe':u,'bindings':dict(bindings,source_root=str(alternate))});assert relocated['catalogue']==bound['catalogue'];assert relocated['input']['universe']==u and relocated['digest']!=bound['digest']
 moved=copy.deepcopy(definition);moved['revision']='v2';moved['locators'][0]['uri']='custom-api:new-source';u2=call('universe_build',moved);assert u2['digest']!=u['digest'];assert call('universe_bind',{'universe':u2,'bindings':bindings})['catalogue']==bound['catalogue']
 for selector,want in [({'op':'all'},5),({'op':'class','values':['gpu']},1),({'op':'date','basis':'model_introduction','from':'2022-01-01','through':'2026-09-14'},0)]:
  d=copy.deepcopy(definition);d['coverage']=seal(dict(coverage,selector=selector));cu=call('universe_build',d);cb=call('universe_bind',{'universe':cu,'bindings':bindings});assert cb['coverage']['counts']['included']==want;assert len(cb['catalogue']['subjects'])==5
 # Conformant declarations do not imply valid selectors or successful extraction.
 d=copy.deepcopy(definition);d['mapping'][0]['field_section']='article#absent';du=call('universe_build',d);call('universe_bind',{'universe':du,'bindings':bindings},False)
 for mutate in [lambda d:d.update(source_root=str(source_root)),lambda d:d.update(kernel_version='latest'),lambda d:d['sources'][0].update(path='../escape'),lambda d:d['mapping'][0].update(source_id='absent'),lambda d:d['profiles'].append(d['profiles'][0]),lambda d:d['locators'][0].update(source_id='unknown')]:
  d=copy.deepcopy(definition);mutate(d);call('universe_build',d,False)
 for b in [dict(bindings,source_root='relative'),dict(bindings,source_root=str(source_root)+'/'),dict(bindings,decoders={'unused':str(root)})]:call('universe_bind',{'universe':u,'bindings':b},False)
 (alternate/sources[0]['path']).write_bytes(b'changed');call('universe_bind',{'universe':u,'bindings':dict(bindings,source_root=str(alternate))},False)
 (alternate/sources[0]['path']).unlink();(alternate/sources[0]['path']).symlink_to(source_root/sources[0]['path']);call('universe_bind',{'universe':u,'bindings':dict(bindings,source_root=str(alternate))},False)
 (alternate/sources[0]['path']).unlink();(alternate/sources[0]['path']).write_bytes((source_root/sources[0]['path']).read_bytes())
 call('universe_bind',{'universe':u,'bindings':bindings})
 if cases:(cases/'SUMMARY.json').write_text(json.dumps({'status':'passed','calls':len(calls),'rejections':sum(not c['good']for c in calls),'classes':[c[0]for c in metrics],'fixture_only':True},indent=2)+'\n')
 print('PASS',len(calls),'calls,',sum(not c['good']for c in calls),'rejections; five caller classes, history, exact replay and relocation')
 return calls

if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--engine');p.add_argument('--qxctl');p.add_argument('--prefix');p.add_argument('--cases');a=p.parse_args()
 if a.cases:
  root=Path(a.cases).resolve();root.mkdir(parents=True,exist_ok=True);run(a.engine,root,a.qxctl,a.prefix,root)
 else:
  with tempfile.TemporaryDirectory(prefix='shv-profile-')as tmp:run(a.engine,Path(tmp).resolve(),a.qxctl,a.prefix)
