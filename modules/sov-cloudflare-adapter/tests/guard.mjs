// Offline qualification of the actual pinned CLI with synthetic fetch responses.
// No Cloudflare account, network request or provider mutation is used here.
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
const [root,bundle]=process.argv.slice(2);
assert(root&&bundle);
const owner=JSON.parse(fs.readFileSync(path.join(root,'knowledge/sov/cloudflare/owner.json')));
const node=path.join(bundle,'node/bin/node'), cli=path.join(bundle,'cf/node_modules/cf/bin/cf');
assert.equal(JSON.parse(fs.readFileSync(path.join(bundle,'bundle.json'))).cf_version,'1.0.0-beta.28');
const scratch=fs.mkdtempSync(path.join(fs.realpathSync(os.tmpdir()),'cf-guard-'));
fs.chmodSync(scratch,0o700);
let processes=0, attempts=0;
try {
 fs.copyFileSync(path.join(root,'modules/sov-cloudflare-adapter/runtime/request-guard.mjs'),path.join(scratch,'guard.mjs'));
 const account='0'.repeat(32),zone='2'.repeat(32);
 const env={PATH:'/usr/bin:/bin',HOME:scratch,XDG_CONFIG_HOME:scratch,XDG_CACHE_HOME:scratch,TMPDIR:scratch,CI:'1',CF_SEND_TELEMETRY:'false',CF_DELEGATION:'1',DO_NOT_TRACK:'1',NO_COLOR:'1',CLOUDFLARE_ACCOUNT_ID:account,CLOUDFLARE_API_TOKEN:'FAKE-CREDENTIAL-SENTINEL'};
 function run(profile,mode='success',mismatch=false){
  const p=profile.template;let route=profile.path.replace('{account_or_zone}','accounts').replace('{account_or_zone_id}',account).replace('{account_id}',account).replace('{zone_id}',zone);
  let args=[...profile.vendor_command];if(profile.resource_parameter){route=route.replace('{'+profile.resource_parameter+'}',p.resource_id);args.push(p.resource_id)}
  if(profile.scope==='zone')args.push('--zone',zone);
  if(p.page)args.push('--page',String(p.page.page),'--per-page',String(p.page.per_page));
  if(p.body)args.push('--body',JSON.stringify(p.body));if(profile.method==='DELETE')args.push('--force');
  const expected={deadline_unix_ms:String(Date.now()+(mode==='expired'?-1:20000)),url:'https://api.cloudflare.com/client/v4'+route+(p.page?'?'+new URLSearchParams(p.page):''),method:profile.method,bodyKind:p.body?'json':'none',...(p.body?{body:p.body}:{})};
  if(mismatch)expected.url+='/wrong-target';fs.writeFileSync(path.join(scratch,'request.json'),JSON.stringify(expected));
  fs.writeFileSync(path.join(scratch,'entry.mjs'),`globalThis.fetch=async()=>{${mode==='throw'?"throw new Error('PRIVATE-ERROR-SENTINEL')":`return new Response(JSON.stringify({success:${mode==='success'},result:${profile.operation==='tunnels.token.get'?"'PRIVATE-TOKEN-SENTINEL'":"{id:'11111111-1111-4111-8111-111111111111'}"},errors:[{message:'PRIVATE-ERROR-SENTINEL'}]}),{status:${mode==='429'?429:mode==='500'?500:200},headers:{'content-type':'application/json'}})`}};await import('./guard.mjs');process.argv=[process.argv[0],${JSON.stringify(cli)},...process.argv.slice(2)];await import(${JSON.stringify('file://'+cli)});`);
  const r=spawnSync(node,['--permission','--allow-net','--allow-fs-read='+path.join(bundle,'cf'),'--allow-fs-read='+scratch,path.join(scratch,'entry.mjs'),...args],{env,cwd:scratch,stdio:['ignore','pipe','pipe','pipe'],timeout:20000,maxBuffer:4*1024*1024});processes++;
  assert(!r.error,profile.operation+': '+r.error);const wire=r.output[3];let events=[];for(let at=0;at<wire.length;){const size=wire.readUInt32BE(at);at+=4;events.push(JSON.parse(wire.subarray(at,at+size)));at+=size;}
  const count=events.filter(x=>x.stage==='dispatch_started').length;assert.equal(count,mismatch||mode==='expired'?0:1,profile.operation+' dispatch '+r.stderr.toString());attempts+=count;
  assert.equal(events.filter(x=>x.stage==='exited').length,1);if(mode==='success'&&!mismatch)assert.equal(r.status,0,profile.operation+': '+r.stderr.toString());
  for(const e of events)if(e.stage==='response_received'&&mode!=='success')assert.equal(e.body,null);
  // Raw CLI stdout/stderr is deliberately discarded by the production runner.
 }
 for(const p of owner.resource_profiles)run(p);
 const mutation=owner.resource_profiles.find(x=>x.operation==='dns.records.create');
 for(const mode of ['429','500','throw','expired'])run(mutation,mode);run(mutation,'success',true);
 console.log(JSON.stringify({qualification:'offline_pinned_cli_guard',profiles:owner.resource_profiles.length,processes,api_attempts:attempts,real_provider_calls:0,retry_enabled:false,checks:['exact URL/method/body','all delete force profiles','429/500/network failure single attempt','wrong target and expired use refused','error bodies excluded']},null,2));
}finally{fs.rmSync(scratch,{recursive:true,force:true})}
