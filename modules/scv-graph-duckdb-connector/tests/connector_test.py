#!/usr/bin/env python3
"""Focused native storage contract tests; no provider/strategy semantic authority.
The graph fixture is an unchanged retained SCV graph from increment12. Test-only
resealed variants isolate mechanical index invariants, not new provider facts.
"""
import argparse, copy, ctypes, fcntl, hashlib, json, os, pathlib, signal, subprocess, sys, tempfile, time, unittest, uuid
P = pathlib.Path(__file__).resolve().parent
VERSION = "0.1.0-dev"
ENGINE = None
LIBRARY = None
EVIDENCE = None

def canonical(v): return json.dumps(v, ensure_ascii=False, sort_keys=True, separators=(',', ':')).encode()
def digest(v): return 'sha256:' + hashlib.sha256(canonical(v)).hexdigest()
def seal(v): return dict(v, digest=digest(v))
def reseal(v): v.pop('digest', None); return seal(v)
def installation(module, role, version, prefix='/fixture/installed'):
    engine = 'symphony-' + (role if module.endswith('-engine') else module)
    return {'Role':role,'ModuleID':module,'EngineID':engine,'Version':version,'Prefix':prefix,
            'ReceiptPath':f'{prefix}/share/symphony/receipts/{module}/{version}/install-receipt.json',
            'ReceiptDigest':'sha256:'+'1'*64,'ReceiptProtocol':'symphony.knowledge.install-receipt.v2',
            'ExecutablePath':f'{prefix}/libexec/symphony/{module}/{version}/{engine}','ExecutableDigest':'sha256:'+'2'*64}

def raw(root, operation, payload, timeout=5):
    now=int(time.time()*1000)
    request={'protocol':'symphony.knowledge.engine-process.v1','request_id':str(uuid.uuid4()),'correlation_id':str(uuid.uuid4()),'operation':operation,'target_engine':'symphony-scv-graph-duckdb-connector','deadline_unix_ms':now+timeout*1000,'payload':payload}
    p=subprocess.run([ENGINE],input=canonical(request),cwd=root,capture_output=True,timeout=timeout+3)
    try: result=json.loads(p.stdout)
    except Exception: raise AssertionError((p.returncode,p.stdout,p.stderr))
    if EVIDENCE:
        key=f'{len(list(EVIDENCE.glob("*-request.json")))+1:03d}-{operation}'
        (EVIDENCE/f'{key}-request.json').write_bytes(canonical(request)+b'\n')
        (EVIDENCE/f'{key}-response.json').write_bytes(p.stdout)
    return p,result

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

class ConnectorTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.root=pathlib.Path(self.temp.name);self.root.chmod(0o700)
        self.graph=json.loads((P/'fixtures/graph.json').read_text())
        self.base={'tops_id':'01993d63-b40d-7000-8000-000000000013','namespace':'native-test','operation_id':'prepare-1','graph':self.graph,
          'owner':installation('scv-engine','scv','0.10.0-dev'),
          'connector':installation('scv-graph-duckdb-connector','scv-graph-duckdb-connector',VERSION),
          'query_time':'2026-09-13T04:30:18Z'}
    def tearDown(self): self.temp.cleanup()
    def call(self,op,payload,okay=True,timeout=5):
        p,r=raw(self.root,op,payload,timeout)
        self.assertEqual(r['outcome']=='ok',okay,(p.returncode,r.get('error'),r.get('outcome')))
        self.assertEqual(p.returncode==0,okay,r)
        if okay:
            v=r['result']; field='descriptor_digest' if op=='inspect' else 'digest'
            unsealed=dict(v);expected=unsealed.pop(field);self.assertEqual(expected,digest(unsealed));return v
        self.assertIsNone(r['result']);self.assertTrue(r['error']['code']);return r
    def status_input(self,base=None): return {k:(base or self.base)[k] for k in ('tops_id','namespace','operation_id')}
    def prepare(self): return self.call('prepare',self.base)
    def commit(self,prepared=None):
        p=prepared or self.prepare();return self.call('commit',dict(self.status_input(),expected_intent_digest=p['intent']['digest']))
    def query_input(self,done,kind='claims',filters=None,limit=128,cursor=None):
        return {k:self.base[k] for k in ('tops_id','namespace')}|{'snapshot_digest':done['snapshot_digest'],'kind':kind,'filters':filters or {},'limit':limit,'cursor':cursor}
    def test_descriptor_does_not_open_database(self):
        d=self.call('inspect',{});self.assertEqual(len(d['operations']),6 if VERSION=="0.1.0-dev" else 8);self.assertEqual(list(self.root.iterdir()),[])
        commit=next(op for op in d['operations'] if op['operation_name']=='commit');self.assertTrue(commit['expected_state_required']);self.assertEqual(commit['administrative_interactions'],['invoke','recover'])
    def test_prepare_reopen_commit_idempotence(self):
        p=self.prepare();self.assertEqual(p['state'],'prepared');self.assertFalse(p['index_verified'])
        self.assertEqual(self.call('status',self.status_input()),p);self.assertEqual(self.prepare(),p)
        self.call('export',{k:v for k,v in self.query_input(p).items() if k in ('tops_id','namespace','snapshot_digest')},False)
        done=self.commit(p);self.assertEqual(done['state'],'committed');self.assertTrue(done['index_verified']);self.assertEqual(self.commit(p),done)
        self.assertEqual(self.prepare(),done);self.assertEqual(self.call('status',self.status_input()),done)
        for path in self.root.iterdir(): self.assertEqual(path.stat().st_mode&0o777,0o600)
    def test_operation_collision_and_expected_digest_reject(self):
        p=self.prepare();changed=copy.deepcopy(self.base);changed['query_time']='2026-09-13T04:30:19Z';self.call('prepare',changed,False)
        self.call('commit',dict(self.status_input(),expected_intent_digest='sha256:'+'0'*64),False)
        self.assertEqual(self.call('status',self.status_input()),p)
    def test_namespaces_and_shared_snapshot_operations(self):
        done=self.commit();other=dict(self.base,operation_id='other');p=self.call('prepare',other);self.assertEqual(p['snapshot_digest'],done['snapshot_digest'])
        other_ns=dict(self.base,namespace='other');p=self.call('prepare',other_ns);self.assertNotEqual(p['snapshot_digest'],done['snapshot_digest'])
        q=self.query_input(done);q['namespace']='other';self.call('query',q,False)
    def test_exact_filters_pagination_and_cursor_binding(self):
        done=self.commit();all_rows=self.call('query',self.query_input(done,'nodes'))
        self.assertEqual(all_rows['matched_count'],len(self.graph['native_nodes']))
        got=[];cursor=None
        while True:
            page=self.call('query',self.query_input(done,'nodes',limit=13,cursor=cursor));got+=page['rows'];cursor=page['next_cursor']
            if cursor is None:break
        self.assertEqual(got,all_rows['rows'])
        first=self.call('query',self.query_input(done,'nodes',limit=1));bad=copy.deepcopy(first['next_cursor']);bad['query_digest']='sha256:'+'f'*64
        self.call('query',self.query_input(done,'nodes',limit=1,cursor=bad),False)
        bad=copy.deepcopy(first['next_cursor']);bad['after_key']='missing';self.call('query',self.query_input(done,'nodes',cursor=bad),False)
        node=self.graph['native_nodes'][0];filtered=self.call('query',self.query_input(done,'nodes',{'node_id':node['node_id'],'kind':node['kind'],'capture_digest':node['capture_digest']}))
        self.assertEqual(filtered['rows'],[{'key':node['node_id'],'value':node}])
    def test_claim_scope_and_structural_edges(self):
        done=self.commit()
        for claim in self.graph['claims']:
            q=self.query_input(done,filters={k:claim[k] for k in ('claim_id','subject','predicate','scope')});r=self.call('query',q);self.assertEqual(r['rows'],[{'key':claim['claim_id'],'value':claim}])
        edge=self.graph['native_edges'][0];r=self.call('query',self.query_input(done,'edges',edge));self.assertEqual(r['rows'],[{'key':digest(edge),'value':edge}])
    def test_opaque_control_ids_and_empty_scope_are_preserved(self):
        self.graph['claims'][0]['claim_id']='opaque\nclaim'
        self.graph['claims'][0]['scope']={'empty':'','nul':'\0'}
        self.base['graph']=reseal(self.graph)
        done=self.commit();claim=self.base['graph']['claims'][0]
        result=self.call('query',self.query_input(done,filters={'claim_id':claim['claim_id'],'scope':claim['scope']}))
        self.assertEqual(result['rows'],[{'key':claim['claim_id'],'value':claim}])
        self.assertEqual(self.call('query',self.query_input(done,filters={'subject':'x'*65536}))['matched_count'],0)
    def test_reject_same_named_index_with_wrong_expression(self):
        self.commit();db=SQL(self.root/'index.duckdb');db.query('DROP INDEX nodes_kind');db.query('CREATE INDEX nodes_kind ON nodes(node_id)');db.close()
        self.call('status',self.status_input(),False)
    def test_reject_index_column_and_inventory_corruption(self):
        self.commit();db=SQL(self.root/'index.duckdb');db.query("UPDATE nodes SET kind='tampered' WHERE row_key=(SELECT min(row_key) FROM nodes)");db.close()
        self.call('status',self.status_input(),False)
    def test_reject_embedded_nul_stored_bytes(self):
        self.commit();db=SQL(self.root/'index.duckdb');db.query("UPDATE nodes SET value=value || chr(0) || 'garbage' WHERE row_key=(SELECT min(row_key) FROM nodes)");db.close()
        self.call('status',self.status_input(),False)
    def test_reject_deleted_projection_row(self):
        done=self.commit();db=SQL(self.root/'index.duckdb');db.query("DELETE FROM edges WHERE row_key=(SELECT min(row_key) FROM edges)");db.close()
        self.call('export',{k:v for k,v in self.query_input(done).items() if k in ('tops_id','namespace','snapshot_digest')},False)
    def test_reject_sealed_intent_database_substitution(self):
        self.prepare();db=SQL(self.root/'index.duckdb');db.query("UPDATE intents SET intent_digest='sha256:"+'0'*64+"'");db.close();self.call('status',self.status_input(),False)
    def test_reject_schema_changes(self):
        self.prepare();db=SQL(self.root/'index.duckdb');db.query('CREATE TABLE unrelated (x VARCHAR)');db.close();self.call('status',self.status_input(),False)
    def test_owned_private_paths_and_symlinks(self):
        self.root.chmod(0o755);self.call('prepare',self.base,False);self.root.chmod(0o700)
        target=self.root/'target';target.write_text('unchanged');target.chmod(0o600);(self.root/'index.duckdb').symlink_to(target)
        self.call('prepare',self.base,False);self.assertEqual(target.read_text(),'unchanged')
    def test_lock_deadline_and_no_missing_read_creation(self):
        self.call('status',self.status_input(),False);self.assertEqual(list(self.root.iterdir()),[])
        self.prepare()
        with open(self.root/'connector.lock','r+b') as f:
            fcntl.flock(f,fcntl.LOCK_EX);start=time.monotonic();self.call('status',self.status_input(),False,timeout=1);self.assertLess(time.monotonic()-start,2.5)
    def test_shape_identity_and_calendar_boundaries(self):
        for patch in [{'query_time':'2026-02-30T00:00:00Z'},{'operation_id':'spaces forbidden'},{'tops_id':'01993d63-b40d-0000-8000-000000000013'},{'namespace':'../escape'}]:
            self.call('prepare',dict(self.base,**patch),False)
        altered=copy.deepcopy(self.base);altered['connector']['Role']='scv';self.call('prepare',altered,False)
        graph=copy.deepcopy(self.graph);graph['native_nodes'].append(graph['native_nodes'][0]);self.call('prepare',dict(self.base,graph=reseal(graph)),False)
        altered=copy.deepcopy(self.base);altered['tops_id']='01993d63-b40d-1000-8000-000000000013';self.call('prepare',altered)
    def test_abrupt_duckdb_transaction_rollback_preserves_committed_projection(self):
        done=self.commit();ready=self.root/'writer-ready'
        process=subprocess.Popen([sys.executable,__file__,'--crash-writer',str(self.root/'index.duckdb'),'--library',LIBRARY,'--ready',str(ready)])
        try:
            for _ in range(300):
                if ready.exists():break
                if process.poll() is not None:self.fail('crash writer exited before readiness')
                time.sleep(.01)
            self.assertTrue(ready.exists());process.kill();self.assertEqual(process.wait(timeout=5),-signal.SIGKILL)
            self.assertEqual(self.call('status',self.status_input()),done)
        finally:
            if process.poll() is None:process.kill();process.wait()

def main():
    global ENGINE,LIBRARY,EVIDENCE,VERSION
    ap=argparse.ArgumentParser();ap.add_argument('--version',default='0.1.0-dev',choices=['0.1.0-dev','0.2.0-dev']);ap.add_argument('--engine');ap.add_argument('--library');ap.add_argument('--evidence');ap.add_argument('--crash-writer');ap.add_argument('--ready');args,rest=ap.parse_known_args()
    VERSION=args.version
    LIBRARY=str(pathlib.Path(args.library).resolve())
    if args.crash_writer:
        db=SQL(pathlib.Path(args.crash_writer));db.query('BEGIN TRANSACTION');db.query('DELETE FROM nodes');pathlib.Path(args.ready).write_text('uncommitted delete completed\n');time.sleep(60);return
    ENGINE=str(pathlib.Path(args.engine).resolve())
    if args.evidence:EVIDENCE=pathlib.Path(args.evidence);EVIDENCE.mkdir(parents=True,exist_ok=False)
    unittest.main(argv=[sys.argv[0],*rest],verbosity=2)
if __name__=='__main__':main()
