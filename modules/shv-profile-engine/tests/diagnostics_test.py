"""Source-failure isolation and artifact-backed reference paths; synthetic fixtures."""
import argparse,copy,hashlib,json,subprocess,tempfile,time,shutil
from pathlib import Path

def canonical(v):return json.dumps(v,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()
def seal(v):v=copy.deepcopy(v);v.pop('digest',None);v['digest']='sha256:'+hashlib.sha256(canonical(v)).hexdigest();return v

def run(engine,root,qxctl=None,prefix=None,version="0.2.0-dev"):
 calls=[]
 def call(op,p,good=True):
  p=copy.deepcopy(p)
  if op=='extraction_diagnose'and p.get('source_root')==str(root/'sources'):
   target=root/f'case-{len(calls):03d}-sources';shutil.copytree(root/'sources',target,symlinks=True);p['source_root']=str(target)
  request={'protocol':'symphony.knowledge.engine-process.v1','request_id':'diagnose','correlation_id':'diagnose','operation':op,'target_engine':'symphony-shv-profile','deadline_unix_ms':int(time.time()*1000)+30000,'payload':p}
  if qxctl:
   route={'extraction_diagnose':['mapping','diagnose-source'],'references_analyze':['references','analyze'],'inspect':['profile','inspect']}[op];path=root/'input.json';path.write_bytes(canonical(p));args=[qxctl,'shv',*route,'--prefix',prefix,'--version',version,'--json'];args+=[]if op=='inspect'else['--input',str(path)];x=subprocess.run(args,capture_output=True)
  else:x=subprocess.run([engine],input=canonical(request),capture_output=True)
  assert (x.returncode==0)==good,(op,good,x.stdout,x.stderr);v=json.loads(x.stdout);r=v.get('result');calls.append({'operation':op,'input':p,'good':good,'result':r});(root/f'{len(calls):03d}.json').write_bytes(canonical(calls[-1]));return r
 assert len(call('inspect',{})['operations'])==7
 sources=root/'sources';sources.mkdir();raw=b'<div id="head"><h1>Fixture</h1></div><article id="spec"><dl><dt>Count</dt><dd>16</dd><dt>Modes</dt><dd>1P / 2P</dd><dt>End</dt><dd>done</dd></dl></article>'
 (sources/'one.html').write_bytes(raw)
 source={'id':'one','path':'one.html','bytes':len(raw),'digest':'sha256:'+hashlib.sha256(raw).hexdigest(),'format':'html'}
 fields=[{'predicate':'count','label':'Count','next_label':'Modes','value_type':'integer','qualifier':'documented'},{'predicate':'modes','label':'Modes','next_label':'End','value_type':'tokens','qualifier':'documented'}]
 mapping={'id':'fixture','source_id':'one','manufacturer':'Fixture','model':'Fixture','hardware_class':'custom','heading_section':'div#head','field_section':'article#spec','fields':fields}
 p={'source_root':str(sources),'sources':[source],'subjects':[mapping]}
 r=call('extraction_diagnose',p);assert r['counts']=={'extracted':2,'failed':0,'unavailable':0}
 for key,value in [('label','absent'),('next_label','absent'),('value_type','date')]:
  bad=copy.deepcopy(p);bad['subjects'][0]['fields'][0][key]=value;r=call('extraction_diagnose',bad);assert r['counts']=={'extracted':1,'failed':1,'unavailable':0};assert r['subjects'][0]['fields'][0]['error']['message']
 for key,value in [('field_section','article#missing'),('heading_section','div#missing'),('field_section','invalid-selector')]:
  bad=copy.deepcopy(p);bad['subjects'][0][key]=value;assert call('extraction_diagnose',bad)['counts']['failed']==2
 bad=copy.deepcopy(p);bad['sources'][0]['digest']='sha256:'+'0'*64;r=call('extraction_diagnose',bad);assert r['sources'][0]['status']=='unverified'and r['counts']['unavailable']==2
 bad=copy.deepcopy(p);bad['sources'].append(dict(source,id='two',path='missing.html'));bad['subjects'].append(dict(mapping,id='missing',source_id='two'));r=call('extraction_diagnose',bad);assert r['counts']=={'extracted':2,'failed':0,'unavailable':2}
 (sources/'link.html').symlink_to(sources/'one.html');bad=copy.deepcopy(p);bad['sources'][0]['path']='link.html';assert call('extraction_diagnose',bad)['counts']['unavailable']==2
 bad=copy.deepcopy(p);bad['subjects'][0]['fields']=[];assert call('extraction_diagnose',bad)['subjects'][0]['fields']==[]
 for mutate in [lambda x:x.update(extra=True),lambda x:x.update(source_root='relative'),lambda x:x['sources'].append(x['sources'][0]),lambda x:x['subjects'][0].update(source_id='unknown'),lambda x:x['subjects'][0]['fields'].append(x['subjects'][0]['fields'][0])]:
  bad=copy.deepcopy(p);mutate(bad);call('extraction_diagnose',bad,False)
 # Explicit scoped table field failure does not hide the adjacent valid scalar.
 table=b'<div id="head"><h1>Table</h1><section id="spec"><table><tr><th>Count</th><td>16</td></tr><tr><th>Launch</th><td>Q1\'23</td></tr></table></section></div>'
 (sources/'table.html').write_bytes(table);s=dict(source,path='table.html',bytes=len(table),digest='sha256:'+hashlib.sha256(table).hexdigest());m=dict(mapping,model='Table',interpretation_profile='scoped_tables.v1');m.pop('field_section');m['fields']=[dict(fields[0],section='section#spec',next_label='Launch'),{'predicate':'model_introduction','section':'section#spec','label':'Launch','next_label':None,'value_type':'quarter_20yy','qualifier':'documented'}]
 tp=dict(p,sources=[s],subjects=[m]);assert call('extraction_diagnose',tp)['counts']['extracted']==2
 bad=copy.deepcopy(tp);bad['subjects'][0]['fields'][0]['section']='section#absent';assert call('extraction_diagnose',bad)['counts']=={'extracted':1,'failed':1,'unavailable':0}
 # Exact reference documents, including retained history and an opaque source.
 digest='sha256:'+hashlib.sha256(raw).hexdigest();snapshot=seal({'protocol':'fixture.snapshot','source_digest':digest});history=seal({'protocol':'fixture.history','entries':[snapshot['digest'],snapshot['digest']]});operation=seal({'protocol':'fixture.operation','history':history['digest']});isolated=seal({'protocol':'fixture.isolated'})
 def obj(id,kind,doc):return {'id':id,'kind':kind,'digest':doc['digest'],'document':doc}
 objects=[obj('snapshot','graph_snapshot',snapshot),obj('history','catalogue_history',history),obj('operation','retained_operation',operation),{'id':'source','kind':'capture','digest':digest,'document':None},obj('isolated','custom',isolated)]
 edges=[{'from':'history','to':'snapshot','pointer':'/entries/0'},{'from':'history','to':'snapshot','pointer':'/entries/1'},{'from':'snapshot','to':'source','pointer':'/source_digest'},{'from':'operation','to':'history','pointer':'/history'}]
 refs={'objects':objects,'edges':edges,'root_ids':['operation'],'candidate_ids':['source','snapshot','isolated']}
 r=call('references_analyze',refs);assert r['candidates'][0]['path']==['operation','history','snapshot','source'];assert len(r['candidates'][1]['incoming_edges'])==2;assert r['candidates'][2]['status']=='not_reachable_in_supplied_graph';assert r['deletion_authorized']==False and r['uninspected_object_ids']==['source']
 multi=copy.deepcopy(refs);multi['root_ids'].append('snapshot');assert call('references_analyze',multi)['candidates'][0]['path']==['snapshot','source']
 empty=copy.deepcopy(refs);empty['root_ids']=[];assert call('references_analyze',empty)['reachable_ids']==[]
 cycle=copy.deepcopy(refs);cycle['edges'].append({'from':'history','to':'history','pointer':'/digest'});assert call('references_analyze',cycle)['reachable_ids']==r['reachable_ids']
 custom=copy.deepcopy(refs);doc={'a/b~c':digest};custom['objects'].append({'id':'custom-root','kind':'user_module','digest':'sha256:'+hashlib.sha256(canonical(doc)).hexdigest(),'document':doc});custom['edges'].append({'from':'custom-root','to':'source','pointer':'/a~1b~0c'});custom['root_ids']=['custom-root'];assert call('references_analyze',custom)['candidates'][0]['path']==['custom-root','source']
 for mutate in [lambda x:x['edges'][0].update(pointer='/entries/2'),lambda x:x['edges'][0].update(pointer='/entries/00'),lambda x:x['edges'][0].update(pointer='/~2bad'),lambda x:x['edges'][0].update(to='isolated'),lambda x:x['edges'].append(x['edges'][0]),lambda x:x['objects'][0].update(digest='sha256:'+'0'*64),lambda x:x['objects'][0]['document'].update(extra=True),lambda x:x['edges'][0].update(**{'from':'source'}),lambda x:x['root_ids'].append('unknown'),lambda x:x.update(deletion_authorized=True)]:
  bad=copy.deepcopy(refs);mutate(bad);call('references_analyze',bad,False)
 (root/'SUMMARY.json').write_text(json.dumps({'status':'passed','calls':len(calls),'rejections':sum(not c['good']for c in calls)},indent=2)+'\n');print('PASS diagnostics/references',len(calls),'calls')

if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--engine');p.add_argument('--qxctl');p.add_argument('--prefix');p.add_argument('--cases');p.add_argument('--version',default='0.2.0-dev');a=p.parse_args()
 if a.cases:
  root=Path(a.cases).resolve();root.mkdir(parents=True,exist_ok=True);run(a.engine,root,a.qxctl,a.prefix,a.version)
 else:
  with tempfile.TemporaryDirectory(prefix='shv-diagnostics-')as temp:run(a.engine,Path(temp).resolve())
