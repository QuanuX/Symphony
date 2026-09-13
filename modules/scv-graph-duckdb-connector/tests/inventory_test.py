#!/usr/bin/env python3
"""Focused v0.2 inventory/planning producer tests; no transfer or deletion API."""
import argparse,copy,json,pathlib,unittest,tempfile
import connector_test as base

class InventoryTests(base.ConnectorTests):
    def setUp(self):
        super().setUp()
        self.base['connector']=base.installation('scv-graph-duckdb-connector','scv-graph-duckdb-connector','0.2.0-dev')
    def inventory(self,revision=None,cursor=None,limit=1,namespace=None,okay=True):
        return self.call('inventory',{'tops_id':self.base['tops_id'],'namespace':namespace or self.base['namespace'],'expected_revision':revision,'cursor':cursor,'limit':limit},okay)
    def plan_input(self,inventory,ids):
        return {'tops_id':self.base['tops_id'],'namespace':self.base['namespace'],'expected_revision':inventory['manifest']['digest'],'operation_ids':ids,
                'source_connector':self.base['connector'],'target_connector':base.installation('scv-graph-duckdb-connector','scv-graph-duckdb-connector','0.2.0-dev',str(self.root/'target-package')),
                'target_root':str(self.root/'not-created-target'),'capacity':{'intents':128,'snapshots':128}}
    def test_inventory_revision_alias_and_pages(self):
        self.commit();a=self.inventory();alias=dict(self.base,operation_id='alias');self.call('prepare',alias)
        b=self.inventory();self.assertNotEqual(a['manifest']['digest'],b['manifest']['digest']);self.assertEqual(len(b['manifest']['snapshots']),1)
        self.assertEqual(b['manifest']['snapshots'][0]['operation_ids'],['alias','prepare-1']);self.assertEqual(b['manifest']['snapshots'][0]['committed_operations'],1)
        self.assertEqual(b['records'][0]['intent']['operation_id'],'alias')
        page=self.inventory(cursor=b['next_cursor']);self.assertEqual(page['manifest'],b['manifest']);self.assertEqual(page['records'][0]['intent']['operation_id'],'prepare-1');self.assertIsNone(page['next_cursor'])
        self.call('prepare',alias);self.assertEqual(self.inventory()['manifest'],b['manifest'])
        self.call('commit',dict(self.status_input(alias),expected_intent_digest=b['records'][0]['intent']['digest']))
        c=self.inventory();self.assertNotEqual(c['manifest']['digest'],b['manifest']['digest']);self.inventory(cursor=b['next_cursor'],okay=False);self.inventory(revision=b['manifest']['digest'],okay=False)
    def test_inventory_other_namespace_revision_without_disclosure(self):
        self.commit();a=self.inventory();other=dict(self.base,namespace='other-secret',operation_id='other-op');self.call('prepare',other)
        b=self.inventory();self.assertNotEqual(a['manifest']['digest'],b['manifest']['digest']);self.assertEqual(a['manifest']['entries'],b['manifest']['entries']);self.assertEqual(b['manifest']['global_counts'],{'intents':2,'snapshots':1})
        self.assertNotIn('other-secret',json.dumps(b));self.assertNotIn('other-op',json.dumps(b));empty=self.inventory(namespace='empty');self.assertEqual(empty['manifest']['entries'],[]);self.assertEqual(empty['records'],[])
    def test_inventory_rejects_orphan_rows_and_snapshots(self):
        self.commit();sql=base.SQL(self.root/'index.duckdb')
        try:sql.query("INSERT INTO nodes SELECT tops_id,'orphan',snapshot_digest,row_key,value,node_id,kind,capture_digest FROM nodes LIMIT 1")
        finally:sql.close()
        self.inventory(okay=False)
        sql=base.SQL(self.root/'index.duckdb')
        try:sql.query("DELETE FROM nodes WHERE namespace='orphan'");sql.query('DELETE FROM intents')
        finally:sql.close()
        self.inventory(okay=False)
    def test_transfer_plan_preserves_selection_and_lineage(self):
        self.commit();alias=dict(self.base,operation_id='alias');self.call('prepare',alias);inv=self.inventory(limit=16)
        before=inv['manifest'];payload=self.plan_input(inv,['prepare-1','alias']);plan=self.call('transfer_plan',payload)
        self.assertEqual([r['source']['intent']['operation_id'] for r in plan['selected']],payload['operation_ids'])
        self.assertEqual(plan['requirements'],{'intents':2,'snapshots':1});self.assertEqual(plan['disposition'],'ready');self.assertEqual(plan['excluded_operation_ids'],[])
        for row in plan['selected']:
            intent=copy.deepcopy(row['source']['intent']);snapshot=intent['snapshot'];snapshot['connector']=payload['target_connector'];intent['snapshot']=base.reseal(snapshot)
            self.assertEqual(row['target_snapshot_digest'],intent['snapshot']['digest']);self.assertEqual(row['target_intent_digest'],base.reseal(intent)['digest'])
        self.assertFalse((self.root/'not-created-target').exists());self.assertEqual(self.inventory()['manifest'],before)
        payload['operation_ids']=['alias'];partial=self.call('transfer_plan',payload);self.assertEqual(partial['requirements'],{'intents':1,'snapshots':0});self.assertEqual(partial['excluded_operation_ids'],['prepare-1'])
    def test_transfer_plan_capacity_and_stale_rejection(self):
        self.commit();inv=self.inventory();p=self.plan_input(inv,['prepare-1']);p['capacity']={'intents':0,'snapshots':0};blocked=self.call('transfer_plan',p)
        self.assertEqual(blocked['blockers'],['intent_capacity','snapshot_capacity']);self.assertEqual(blocked['disposition'],'blocked')
        for change in [{'operation_ids':['missing']},{'operation_ids':['prepare-1','prepare-1']},{'expected_revision':'sha256:'+'0'*64},{'capacity':{'intents':129,'snapshots':128}}]:self.call('transfer_plan',dict(p,**change),False)
        self.assertEqual(self.inventory()['manifest'],inv['manifest'])
    def test_inventory_shapes_and_missing_store(self):
        self.inventory(okay=False);self.assertEqual(list(self.root.iterdir()),[])
        self.commit()
        for limit in [0,17,1.5,4294967297,9007199254740991]:self.inventory(limit=limit,okay=False)
        self.inventory(cursor={'revision':'sha256:'+'0'*64,'after_operation_id':'prepare-1'},okay=False)
    def test_transfer_plan_rejects_unsupported_target(self):
        self.commit();p=self.plan_input(self.inventory(),['prepare-1']);p['target_connector']['Version']='9.0.0';self.call('transfer_plan',p,False)

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--engine',required=True);parser.add_argument('--library',required=True);parser.add_argument('--evidence',type=pathlib.Path);args=parser.parse_args()
    base.ENGINE=str(pathlib.Path(args.engine).resolve());base.LIBRARY=str(pathlib.Path(args.library).resolve());temporary=tempfile.TemporaryDirectory() if args.evidence is None else None; base.EVIDENCE=args.evidence.resolve() if args.evidence else pathlib.Path(temporary.name)/"evidence";base.EVIDENCE.mkdir(parents=True,exist_ok=False)
    names=[n for n in InventoryTests.__dict__ if n.startswith('test_')];suite=unittest.TestSuite(InventoryTests(n) for n in names)
    result=unittest.TextTestRunner(verbosity=2).run(suite);raise SystemExit(not result.wasSuccessful())
