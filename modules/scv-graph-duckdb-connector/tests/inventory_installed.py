#!/usr/bin/env python3
"""Installed inventory/transfer-plan acceptance, with explicit legacy compatibility.
No transfer is executed and no caller data is deleted. Private fixtures are owned
by this campaign. Exact SCV owner replay uses a recorded simulated query time.
"""
import argparse,json
from pathlib import Path
from installed_integration import Campaign,canonical

class InventoryCampaign(Campaign):
    def inv(self,name,revision=None,cursor=None,limit=1,**kw):
        return self.qx(name,'inventory',{'expected_revision':revision,'cursor':cursor,'limit':limit},**kw)
    def plan(self,name,revision,ids,target=None,capacity=None,**kw):
        return self.qx(name,'transfer-plan',{'expected_revision':revision,'operation_ids':ids,'target':target or self.target,'capacity':capacity or {'intents':128,'snapshots':128}},**kw)
    def test_inventory_and_transfer_planning(self):
        self.summary['recovery_scope']='Inventory and planning only; no transfer, retirement or migration execution.'
        target=self.out/'empty-target';target.mkdir(mode=0o700)
        self.target={'prefix':str(self.args.connector_prefix.resolve()),'version':'0.2.0-dev','root':str(target)}
        first=self.qx('initial-import','import',{'operation_id':'original','graph':self.graph,'query_time':self.time})['connector_result'];snapshot=first['intent']['snapshot']
        a=self.inv('inventory-one')['connector_result'];revision=a['manifest']['digest']
        self.check(len(a['manifest']['entries'])==1 and a['records'][0]==first,'Inventory full record matches exact installed import')
        alias={'tops_id':self.tops,'namespace':self.namespace,'operation_id':'alias','graph':self.graph,'query_time':self.time,'owner':snapshot['owner'],'connector':snapshot['connector']}
        self.native('prepare-alias','prepare',alias,snapshot['connector'])
        b=self.inv('inventory-prepared')['connector_result'];self.check(b['manifest']['digest']!=revision and b['manifest']['snapshots'][0]['committed_operations']==1,'Prepared alias changes revision without a second published snapshot')
        self.inv('stale-observation',revision=revision,ok=False)
        page=self.inv('inventory-page-two',cursor=b['next_cursor'])['connector_result'];self.check(page['manifest']==b['manifest'] and page['records'][0]['intent']['operation_id']=='original','Inventory pages retain one exact revision and byte-sorted record order')
        self.qx('recover__alias','recover');c=self.inv('inventory-committed')['connector_result'];revision=c['manifest']['digest']
        self.check(c['manifest']['snapshots'][0]['committed_operations']==2 and len(c['manifest']['snapshots'])==1,'Committed aliases preserve a shared snapshot reference group')
        self.inv('stale-cursor',cursor=b['next_cursor'],ok=False)
        plan=self.plan('same-install-plan',revision,['alias','original'])
        self.check(plan['disposition']=='ready' and len(plan['owner_evaluations'])==2 and all(v['outcome']=='validated' for v in plan['owner_evaluations']),'Ready plan retains native owner evaluation for every selected operation')
        body=plan['connector_result'];self.check(body['requirements']=={'intents':2,'snapshots':1} and all(x['target_snapshot_digest']==snapshot['digest'] for x in body['selected']),'Same installation preserves snapshot identity and deduplicates required published snapshots')
        legacy_target=dict(self.target,prefix=str(self.args.legacy_prefix.resolve()),version='0.1.0-dev')
        mapped=self.plan('legacy-target-plan',revision,['original'],target=legacy_target)['connector_result']
        self.check(mapped['selected'][0]['target_snapshot_digest']!=snapshot['digest'] and mapped['excluded_operation_ids']==['alias'],'Different exact target installation creates explicit new identity and excludes unselected aliases')
        blocked=self.plan('capacity-blockers',revision,['alias','original'],capacity={'intents':0,'snapshots':0})
        self.check(blocked['disposition']=='blocked' and [x['code'] for x in blocked['blockers']]==['intent_capacity','snapshot_capacity'],'Caller capacity limits are explicit plan blockers')
        (target/'caller-marker').write_text('retained target bytes')
        occupied=self.plan('nonempty-target',revision,['original']);self.check(occupied['disposition']=='blocked' and any(x['code']=='target_not_empty' for x in occupied['blockers']),'Nonempty target is reported without modifying it')
        self.check((target/'caller-marker').read_text()=='retained target bytes' and len(list(target.iterdir()))==1,'Planning creates no target database, lock or receipt')
        (target/'caller-marker').unlink() # Remove only this campaign-created marker.
        self.plan('missing-selection',revision,['missing'],ok=False)
        self.plan('unsupported-target',revision,['original'],target=dict(self.target,version='9.0.0'),ok=False)
        self.check(self.inv('source-after-plans')['connector_result']['manifest']==c['manifest'],'Successful and blocked plans preserve complete logical source inventory')
        self.qx('other-namespace','import',{'operation_id':'private-other','graph':self.graph,'query_time':self.time},namespace='private-other-namespace')
        d=self.inv('cross-namespace-inventory')['connector_result'];self.check(d['manifest']['entries']==c['manifest']['entries'] and d['manifest']['digest']!=revision,'Other namespace activity invalidates revision without disclosing its records')
        self.plan('cross-namespace-stale-plan',revision,['original'],ok=False)
        self.check('private-other' not in json.dumps(d),'Scoped inventory omits other namespace identities')
        owner=self.clone_owner();imported=self.qx('copy-owner-import','import',{'operation_id':'copy-owner','graph':self.graph,'query_time':self.time},owner=owner)['connector_result']
        executable=Path(imported['intent']['snapshot']['owner']['ExecutablePath']);hidden=executable.with_suffix('.hidden');executable.rename(hidden)
        try:
            observed=self.inv('inventory-without-owner')['connector_result'];missing=self.plan('plan-without-owner',observed['manifest']['digest'],['copy-owner'])
            self.check(missing['disposition']=='blocked' and missing['owner_evaluations'][0]['outcome']=='unavailable','Missing exact owner permits inventory and explicitly blocks semantic transfer planning')
        finally:hidden.rename(executable)
        # An old stored snapshot is observed by a new reader without relabeling.
        selected_prefix=self.args.connector_prefix;selected_version=self.connector_version;selected_root=self.root;selected_namespace=self.namespace
        self.root=self.out/'legacy-index';self.root.mkdir(mode=0o700);self.namespace='legacy';self.args.connector_prefix=self.args.legacy_prefix;self.connector_version='0.1.0-dev'
        old=self.qx('legacy-import','import',{'operation_id':'legacy','graph':self.graph,'query_time':self.time})['connector_result']
        self.inv('legacy-inventory-rejected',ok=False)
        self.args.connector_prefix=selected_prefix;self.connector_version=selected_version
        old_inventory=self.inv('new-reader-legacy-inventory')['connector_result'];self.check(old_inventory['records'][0]==old,'New reader preserves exact old connector and intent identities')
        self.qx('new-reader-legacy-export-rejected','export',{'snapshot_digest':old['snapshot_digest'],'query_time':self.time},ok=False)
        self.args.connector_prefix=self.args.legacy_prefix;self.connector_version='0.1.0-dev'
        exported=self.qx('exact-legacy-export','export',{'snapshot_digest':old['snapshot_digest'],'query_time':self.time});self.check(exported['connector_result']['snapshot']==old['intent']['snapshot'],'Exact legacy data operations remain usable after new inventory observation')
        self.args.connector_prefix=selected_prefix;self.connector_version=selected_version;self.root=selected_root;self.namespace=selected_namespace
        self.check(not list(target.iterdir()),'Destination remains empty after all planning')
        self.summary.update(status='passed',transfers_executed=0,source_records_deleted=0,target_databases_created=0);self.save()

if __name__=='__main__':
    p=argparse.ArgumentParser()
    for name in ['qxctl','connector-prefix','owner-prefix','graph','out','legacy-prefix']:p.add_argument('--'+name,type=Path,required=True)
    args=p.parse_args();args.connector_version='0.2.0-dev';campaign=InventoryCampaign(args)
    try:campaign.test_inventory_and_transfer_planning()
    except Exception:campaign.summary['status']='failed';campaign.save();raise
    print(json.dumps({'status':'passed','calls':len(campaign.calls),'assertions':len(campaign.assertions)}))
