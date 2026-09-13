"""Generate the finite SHV source schema; generic graph/descriptor shapes are reused verbatim."""
from pathlib import Path
import json, copy
B=Path(__file__).resolve().parent.parent; R=B.parent.parent
G=json.loads((R/'modules/shv-graph-adapter/schemas/v1/graph-adapter.schema.json').read_text())['$defs']
D={k:copy.deepcopy(G[k]) for k in ['Node','Edge','Graph','token','version','digest','featureId','operationId','operation','interaction','limits','Descriptor']}
def obj(p):return {'type':'object','properties':p,'required':list(p),'additionalProperties':False}
def txt(n=4096):return {'type':'string','minLength':1,'maxLength':n}
def arr(x,n,minimum=0,unique=False):return {'type':'array','items':x,'maxItems':n,'minItems':minimum,**({'uniqueItems':True} if unique else {})}
def ref(n):return {'$ref':'#/$defs/'+n}
def optional(x):return {'anyOf':[x,{'type':'null'}]}
def sealed(proto,p):return obj({'protocol':{'const':proto},**p,'digest':digest})
id={'type':'string','pattern':'^[A-Za-z0-9._-]{1,128}$'};digest={'type':'string','pattern':'^sha256:[0-9a-f]{64}$'}
uri={**txt(),'pattern':'^https?://[A-Za-z0-9][A-Za-z0-9.-]*(?::[1-9][0-9]{0,4})?(?:[/?][!-~]*)?$'}
D['Locator']=obj({'id':id,'uri':uri,'format':{'enum':['html','opaque']}})
D['SourceDefinition']=obj({'source_id':id,'publisher':txt(256),'authority_role':txt(256),'subject_ids':arr(id,32,1,True),'locators':arr(ref('Locator'),16,1)})
D['Source']=sealed('symphony.shv.source-revision.v1',{'definition':ref('SourceDefinition'),'generation':{'type':'integer','minimum':1,'maximum':32},'previous_digest':optional(digest)})
D['InspectInput']=obj({})
D['SourcePlanInput']=obj({'operation_id':id,'current':optional(ref('Source')),'desired':ref('SourceDefinition'),'reason':txt()})
D['SourcePlan']=sealed('symphony.shv.source-plan.v1',{'operation_id':id,'expected_state_digest':optional(digest),'change_kind':{'enum':['onboard','authority_change','relocation']},'reason':txt(),'source':ref('Source')})
D['SourceReduceInput']=obj({'current':optional(ref('Source')),'plan':ref('SourcePlan')})
D['SourceTransition']=sealed('symphony.shv.source-transition.v1',{'operation_id':id,'expected_state_digest':optional(digest),'source':ref('Source')})
D['SourceStatusInput']=obj({'history':arr(ref('Source'),32,1)})
D['SourceStatus']=sealed('symphony.shv.source-status.v1',{'source':ref('Source'),'history_digests':arr(digest,32,1,True)})
D['Manifest']=copy.deepcopy(json.loads((R/'modules/shv-engine/schemas/v1/shv.schema.json').read_text())['$defs']['Source'])
D['UpstreamRevision']=obj({'scheme':txt(128),'value':txt(512)})
capture={'source':ref('Source'),'locator_id':id,'resolved_uri':uri,'redirect_chain':arr(uri,8,0,True),'observed_at':{'type':'string','pattern':'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'},'upstream_revision':optional(ref('UpstreamRevision')),'manifest':ref('Manifest'),'completeness':{'enum':['complete','partial']},'issues':arr(txt(512),32,0,True)}
D['CaptureImportInput']=obj({'source_root':txt(),**capture})
D['Capture']=sealed('symphony.shv.source-capture.v1',capture)
D['CaptureCompareInput']=obj({'source_root':txt(),'previous':ref('Capture'),'current':ref('Capture')})
D['CaptureComparison']=sealed('symphony.shv.capture-comparison.v1',{'previous_digest':digest,'current_digest':digest,'same_logical_source':{'type':'boolean'},'changes':arr({'enum':['source_identity','source_revision','body','representation','acquisition_route','upstream_revision','completeness','observation_time','manifest_identity']},9,0,True)})
D['Bundle']=sealed('symphony.shv.source-bundle.v1',{'captures':arr(ref('Capture'),8)})
D['GraphProjectInput']=obj({'source_root':txt(),'captures':arr(ref('Capture'),8)})
D['GraphValidateInput']=obj({'source_root':txt(),'graph':ref('Graph')})
D['GraphValidation']=sealed('symphony.shv.source-graph-validation.v1',{'graph_digest':digest,'bundle_digest':digest,'valid':{'const':True}})
schema={'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'symphony.shv.source-schema.v1','$comment':'Strict structural discovery. Runtime additionally checks UTF-8 byte lengths, finite URI syntax/port bounds, exact seals, lineage, uniqueness by key, routes/calendar, consumed file bytes and graph correspondence. Generic graph/descriptor definitions copied verbatim.','$defs':D}
(B/'schemas/v1/source.schema.json').write_text(json.dumps(schema,indent=2)+'\n')
templates={'inspect':{},'source_plan':{'operation_id':None,'current':None,'desired':None,'reason':None},'source_reduce':{'current':None,'plan':None},'source_status':{'history':[]},'capture_import':{'source_root':None,**{k:([] if k in ['redirect_chain','issues'] else None) for k in capture}},'capture_compare':{'source_root':None,'previous':None,'current':None},'graph_project':{'source_root':None,'captures':[]},'graph_validate':{'source_root':None,'graph':None}}
(B/'schemas/v1/source.templates.json').write_text(json.dumps(templates,indent=2)+'\n')
print('Generated',len(D),'definitions;',len(templates),'unanswered templates')
