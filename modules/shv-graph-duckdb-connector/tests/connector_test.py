#!/usr/bin/env python3
import argparse,copy,ctypes,hashlib,json,os,pathlib,subprocess,tempfile,time,unittest,uuid
ENGINE=None;LIBRARY=None
canonical=lambda v:json.dumps(v,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()
def seal(v):return dict(v,digest='sha256:'+hashlib.sha256(canonical(v)).hexdigest())
def graph():
 a=seal({'protocol':'caller.example.v1','future_field':{'class':'networking-card','retired':None}})
 return seal({'protocol':'symphony.graph.exchange.v1','owner':{'engine_id':'caller-owner','engine_version':'1','artifact_protocol':a['protocol'],'artifact_digest':a['digest']},'owner_artifact':a,'nodes':[{'id':str(i),'labels':['component'],'properties':{'unknown_metric':{'unit':'caller','value':i}}} for i in range(3)],'edges':[{'id':str(i),'from':'0','to':'1','label':'caller-link','properties':{'future':[i]}} for i in range(3)]})
def installation():
 m='shv-graph-duckdb-connector';v='0.2.0-dev';p='/fixture';return {'Role':m,'ModuleID':m,'EngineID':'symphony-'+m,'Version':v,'Prefix':p,'ReceiptPath':p+'/share/symphony/receipts/'+m+'/'+v+'/install-receipt.json','ReceiptProtocol':'symphony.knowledge.install-receipt.v2','ReceiptDigest':'sha256:'+'1'*64,'ExecutablePath':p+'/libexec/symphony/'+m+'/'+v+'/symphony-'+m,'ExecutableDigest':'sha256:'+'2'*64}
def request(op,p):return {'protocol':'symphony.knowledge.engine-process.v1','request_id':str(uuid.uuid4()),'correlation_id':str(uuid.uuid4()),'target_engine':'symphony-shv-graph-duckdb-connector','operation':op,'deadline_unix_ms':int(time.time()*1000)+20000,'payload':p}
def raw(root,op,p):
 x=subprocess.run([ENGINE],cwd=root,input=canonical(request(op,p)),capture_output=True,timeout=25);r=json.loads(x.stdout);return x,r
class SQL:
    """Test-only direct database corruption/crash oracle through selected DuckDB C ABI."""
    def __init__(self,path):
        self.lib=ctypes.CDLL(LIBRARY); self.db=ctypes.c_void_p();self.con=ctypes.c_void_p()
        self.lib.duckdb_open.argtypes=[ctypes.c_char_p,ctypes.POINTER(ctypes.c_void_p)]
        self.lib.duckdb_connect.argtypes=[ctypes.c_void_p,ctypes.POINTER(ctypes.c_void_p)]
        self.lib.duckdb_query.argtypes=[ctypes.c_void_p,ctypes.c_char_p,ctypes.c_void_p]
        self.lib.duckdb_disconnect.argtypes=[ctypes.POINTER(ctypes.c_void_p)];self.lib.duckdb_close.argtypes=[ctypes.POINTER(ctypes.c_void_p)]
        assert self.lib.duckdb_open(str(path).encode(),ctypes.byref(self.db))==0
        assert self.lib.duckdb_connect(self.db,ctypes.byref(self.con))==0
    def query(self,sql): assert self.lib.duckdb_query(self.con,sql.encode(),None)==0,sql
    def close(self): self.lib.duckdb_disconnect(ctypes.byref(self.con));self.lib.duckdb_close(ctypes.byref(self.db))


class StoreTests(unittest.TestCase):
 def setUp(self):
  self.temp=tempfile.TemporaryDirectory();self.root=pathlib.Path(self.temp.name);self.root.chmod(0o700);self.scope={'tops_id':'01993d63-b40d-7000-8000-000000000013','namespace':'caller'};self.key={**self.scope,'operation_id':'one'};self.p={**self.key,'graph':graph(),'connector':installation()}
 def tearDown(self):self.temp.cleanup()
 def call(self,op,p,ok=True):
  x,r=raw(self.root,op,p);self.assertEqual(x.returncode==0,ok,r);self.assertEqual(r['outcome']=='ok',ok,r);return r['result'] if ok else r
 def commit(self):
  p=self.call('prepare',self.p);c={**self.key,'expected_intent_digest':p['intent']['digest']};r=self.call('commit',c);return r,c
 def test_reopen_retry_export_and_pagination(self):
  prepared=self.call('prepare',self.p);self.assertEqual(prepared,self.call('prepare',self.p));self.call('export',{**self.scope,'snapshot_digest':prepared['snapshot_digest']},False)
  r,c=self.commit();self.assertEqual(r,self.call('commit',c));self.assertEqual(r,self.call('status',self.key));e=self.call('export',{**self.scope,'snapshot_digest':r['snapshot_digest']});self.assertEqual(e['snapshot']['graph'],self.p['graph'])
  q={**self.scope,'snapshot_digest':r['snapshot_digest'],'kind':'edges','filters':{},'cursor':None,'limit':1};one=self.call('query',q);self.assertEqual(one['matched_count'],3);q['cursor']=one['next_cursor'];q['limit']=2;two=self.call('query',q);self.assertEqual(len(two['rows']),2);self.assertIsNone(two['next_cursor']);q['filters']={'label':'different'};self.call('query',q,False)
 def test_operation_conflict_and_expected_digest(self):
  r=self.call('prepare',self.p);p=copy.deepcopy(self.p);p['graph']['nodes'][0]['properties']['new']=True;p['graph'].pop('digest');p['graph']=seal(p['graph']);self.call('prepare',p,False);self.call('commit',{**self.key,'expected_intent_digest':'sha256:'+'0'*64},False);self.assertEqual(self.call('status',self.key)['state'],'prepared')
 def test_scope_isolation(self):
  r,_=self.commit();self.call('status',{**self.key,'namespace':'another'},False);self.call('export',{**self.scope,'namespace':'another','snapshot_digest':r['snapshot_digest']},False)
 def test_complete_inventory_corruption(self):
  r,_=self.commit();db=SQL(self.root/'index.duckdb');db.query("UPDATE nodes SET value='{}' WHERE row_key='2'");db.close();self.call('query',{**self.scope,'snapshot_digest':r['snapshot_digest'],'kind':'nodes','filters':{'id':'0'},'cursor':None,'limit':1},False);self.call('status',self.key,False)
 def test_extra_row_and_schema(self):
  r,_=self.commit();db=SQL(self.root/'index.duckdb');db.query("INSERT INTO nodes SELECT tops_id,namespace,snapshot_digest,'extra',value,'extra' FROM nodes LIMIT 1");db.close();self.call('export',{**self.scope,'snapshot_digest':r['snapshot_digest']},False)
 def test_unknown_schema(self):
  self.commit();db=SQL(self.root/'index.duckdb');db.query('CREATE TABLE unexpected(value VARCHAR)');db.close();self.call('status',self.key,False)
 def test_graph_tamper_and_dangling_endpoint(self):
  self.p['graph']['owner_artifact']['future_field']['new']=1;self.call('prepare',self.p,False);self.p['graph']=graph();self.p['graph']['edges'][0]['to']='absent';self.p['graph'].pop('digest');self.p['graph']=seal(self.p['graph']);self.call('prepare',self.p,False)
 def test_read_does_not_create_and_private_mode(self):
  self.call('status',self.key,False);self.assertFalse((self.root/'index.duckdb').exists());self.root.chmod(0o755);self.call('prepare',self.p,False);self.assertFalse((self.root/'index.duckdb').exists());self.root.chmod(0o700)
 def test_symlink_rejection(self):
  outside=self.root/'elsewhere';outside.write_text('caller');(self.root/'index.duckdb').symlink_to(outside);self.call('prepare',self.p,False);self.assertEqual(outside.read_text(),'caller')
 def test_empty_graph(self):
  g=self.p['graph'];g['nodes']=[];g['edges']=[];g.pop('digest');self.p['graph']=seal(g);r,_=self.commit();q=self.call('query',{**self.scope,'snapshot_digest':r['snapshot_digest'],'kind':'nodes','filters':{},'cursor':None,'limit':1});self.assertEqual(q['rows'],[]);self.assertEqual(q['matched_count'],0)
 def inventory(self,**extra):return self.call('inventory',{**self.scope,'expected_revision':None,'cursor':None,'limit':1,**extra})
 def test_inventory_references_and_revision(self):
  r,_=self.commit();p={**self.p,'operation_id':'two'};second=self.call('prepare',p)
  first=self.inventory();self.assertEqual(len(first['manifest']['entries']),2);self.assertEqual(first['manifest']['snapshots'][0]['committed_operations'],1);self.assertEqual(first['manifest']['global_counts'],{'intents':2,'snapshots':1})
  page=self.inventory(cursor=first['next_cursor'],expected_revision=first['manifest']['digest']);self.assertEqual(page['records'][0]['state'],'prepared');self.assertIsNone(page['next_cursor'])
  self.call('commit',{**self.scope,'operation_id':'two','expected_intent_digest':second['intent']['digest']});self.call('inventory',{**first['input'],'cursor':first['next_cursor']},False)
  self.assertEqual(self.inventory()['manifest']['snapshots'][0]['committed_operations'],2)
 def test_inventory_other_scope_invalidates_without_leaking(self):
  self.commit();first=self.inventory();self.call('prepare',{**self.p,'namespace':'private-other'})
  self.call('inventory',{**first['input'],'expected_revision':first['manifest']['digest']},False)
  current=self.inventory();self.assertEqual(len(current['manifest']['entries']),1);self.assertEqual(current['manifest']['global_counts']['intents'],2)
  empty=self.inventory(namespace='empty');self.assertEqual(empty['records'],[]);self.assertEqual(empty['manifest']['snapshots'],[])
 def test_inventory_orphan_rows_in_other_scope(self):
  self.commit();db=SQL(self.root/'index.duckdb');db.query("INSERT INTO nodes SELECT tops_id,'orphan',snapshot_digest,row_key,value,id FROM nodes LIMIT 1");db.close();self.call('inventory',{**self.scope,'expected_revision':None,'cursor':None,'limit':1},False)
 def test_inventory_missing_database_and_bad_cursor(self):
  q={**self.scope,'expected_revision':None,'cursor':None,'limit':1};self.call('inventory',q,False);self.assertFalse((self.root/'index.duckdb').exists());self.commit()
  first=self.inventory();self.call('inventory',{**q,'cursor':{'revision':first['manifest']['digest'],'after_operation_id':'absent'}},False)
  for value in [0,17,True,1.5]:self.call('inventory',{**q,'limit':value},False)

if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--engine',required=True);p.add_argument('--library',required=True);p.add_argument('--version');args,rest=p.parse_known_args();ENGINE=str(pathlib.Path(args.engine).resolve());LIBRARY=args.library;unittest.main(argv=[__file__,*rest])
