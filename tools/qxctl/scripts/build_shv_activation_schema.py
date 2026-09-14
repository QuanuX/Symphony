"""Build CLI-owned activation schemas with attributed exact owner definitions."""
from pathlib import Path
import json,copy,hashlib
R=Path(__file__).resolve().parents[3]
def obj(p):return {'type':'object','properties':p,'required':list(p),'additionalProperties':False}
def ref(n):return {'$ref':'#/$defs/'+n}
def arr(v,n):return {'type':'array','items':v,'maxItems':n}
def nullable(v):return {'anyOf':[v,{'type':'null'}]}
source=R/'modules/shv-source-engine/schemas/v1/source.schema.json'
D=copy.deepcopy(json.loads(source.read_bytes())['$defs'])
ident={'type':'string','pattern':'^[A-Za-z0-9._-]{1,128}$'}
digest={'type':'string','pattern':'^sha256:[0-9a-f]{64}$'}
uuid={'type':'string','pattern':'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'}
D['ActivationInstallation']=obj({k:{'type':'string','minLength':1} for k in ['Role','ModuleID','EngineID','Version','Prefix','ReceiptPath','ReceiptDigest','ReceiptProtocol','ExecutablePath','ExecutableDigest']})
D['ActivationAuthorization']=json.loads((R/'knowledge/ssiag/schemas/v1/authorization-decision.schema.json').read_bytes())
D['ActivationCapability']=json.loads((R/'knowledge/ssiag/schemas/v1/capability.schema.json').read_bytes())
D['ActivationIntent']=obj({'operation_id':ident,'plan':ref('SourcePlan'),'transition':ref('SourceTransition'),'installation':ref('ActivationInstallation'),'digest':digest})
D['ActivationAttempt']=obj({'intent':ref('ActivationIntent'),'status':{'enum':['prepared','authorized','committed']},'correlation_id':uuid,'authorization':nullable(ref('ActivationAuthorization')),'prior_authorizations':arr(ref('ActivationAuthorization'),64)})
D['ActivationProposalInput']=obj({'operation_id':ident,'desired':ref('SourceDefinition'),'reason':{'type':'string','minLength':1,'maxLength':4096}})
D['ActivationStore']=obj({'protocol':{'const':'symphony.qxctl.shv-source-store.v1'},'tops_id':uuid,'source_id':ident,'source':nullable(ref('Source')),'state_digest':nullable(digest),'head_operation_id':nullable(ident),'operations':{'type':'object','propertyNames':ident,'maxProperties':128,'additionalProperties':ref('ActivationAttempt')},'digest':digest})
D['ActivationResult']=obj({'protocol':{'const':'symphony.qxctl.shv-source-activation-result.v1'},'operation':{'enum':['apply','recover','status']},'tops_id':uuid,'source_id':ident,'source':nullable(ref('Source')),'state_digest':nullable(digest),'head_operation_id':nullable(ident),'history':arr(ref('Source'),32),'attempt':nullable(ref('ActivationAttempt')),'owner_result':{'anyOf':[{'type':'null'},ref('SourceTransition'),ref('SourceStatus')]},'canonical_apply_enabled':{'const':False},'authorization_audit':{'const':'ssiag_policy_decision_only'},'source_write_stav_receipt':{'type':'null'},'digest':digest})
D['ActivationSchema']=obj({'protocol':{'const':'symphony.qxctl.shv-source-activation-schema.v1'},'installation':ref('ActivationInstallation'),'origin':{'const':'qxctl_embedded'},'schema':{'type':'object'},'digest':digest})
D['ActivationTemplate']=obj({'protocol':{'const':'symphony.qxctl.shv-source-activation-template.v1'},'installation':ref('ActivationInstallation'),'operation':{'enum':['propose','apply']},'template':{'type':'object'},'status':{'const':'unanswered_template_not_validated_input'},'digest':digest})
D['ActivationError']=obj({'protocol':{'const':'symphony.qxctl.error.v1'},'outcome':{'const':'error'},'command_id':nullable({'type':'string'}),'error':obj({'code':{'type':'string'},'message':{'type':'string'},'engine_code':nullable({'type':'string'})}),'exit_code':{'type':'integer','minimum':1,'maximum':125}})
doc={'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'symphony.qxctl.shv-source-activation-schema-document.v1','$comment':'CLI-owned activation definitions. Source definitions copied from SHV source engine0.1.0-dev (sha256:'+hashlib.sha256(source.read_bytes()).hexdigest()+'); SSIAG decision/capability schemas retain their owner IDs. Shape checks do not authenticate authority, validate seals or replace source reduction.','$defs':D}
(R/'tools/qxctl/cmd/qxctl/shv_activation.schema.json').write_text(json.dumps(doc,indent=2)+'\n')
