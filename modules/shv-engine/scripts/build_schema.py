from pathlib import Path
import json,copy
R=Path(__file__).resolve().parents[3];B=R/'modules/shv-engine/schemas/v1'
def obj(p):return {'type':'object','properties':p,'required':list(p),'additionalProperties':False}
def arr(v,n):return {'type':'array','items':v,'maxItems':n}
def ref(s):return {'$ref':'#/$defs/'+s}
def txt(n=4096):return {'type':'string','minLength':1,'maxLength':n}
id={'type':'string','pattern':'^[A-Za-z0-9._-]{1,128}$'};digest={'type':'string','pattern':'^sha256:[0-9a-f]{64}$'};date={'type':'string','pattern':'^[0-9]{4}-[0-9]{2}-[0-9]{2}$'}
def sealed(proto,fields):return obj({'protocol':{'const':proto},**fields,'digest':digest})
D={}
D['Value']={'oneOf':[txt(),{'type':'integer','minimum':-9007199254740991,'maximum':9007199254740991},{**arr(txt(128),32),'uniqueItems':True}]}
D['DateInterval']=obj({'from':date,'through':date})
base={'id':id,'manufacturer':txt(256),'model':txt(256),'hardware_class':id,'introduced':{'anyOf':[ref('DateInterval'),{'type':'null'}]}}
D['SubjectSummary']=obj(base)
D['Source']=obj({'id':id,'path':txt(),'bytes':{'type':'integer','minimum':0,'maximum':1048576},'digest':digest,'format':{'enum':['html','opaque']}})
D['MappingField']=obj({'predicate':id,'label':txt(256),'next_label':txt(256),'value_type':{'enum':['string','integer','date','tokens']},'qualifier':txt(256)})
D['SubjectSpec']=obj({'id':id,'manufacturer':txt(256),'model':txt(256),'hardware_class':id,'source_id':id,'heading_section':dict(txt(128),pattern=r'^[a-z][a-z0-9-]*#[^#]+$'),'field_section':dict(txt(128),pattern=r'^[a-z][a-z0-9-]*#[^#]+$'),'fields':arr(ref('MappingField'),16)})
D['Assertion']=obj({'predicate':id,'value':ref('Value'),'qualifier':txt(256),'source_id':id})
D['Subject']=obj({**base,'assertions':arr(ref('Assertion'),16)})
D['Selector']={'oneOf':[obj({'op':{'const':'all'}}),obj({'op':{'const':'date'},'basis':{'const':'model_introduction'},'from':date,'through':date}),obj({'op':{'enum':['ids','class']},'values':{**arr(id,128),'uniqueItems':True}}),obj({'op':{'const':'manufacturer'},'values':{**arr(txt(256),128),'uniqueItems':True}}),obj({'op':{'enum':['and','or']},'args':{**arr(ref('Selector'),32),'minItems':1}}),obj({'op':{'const':'not'},'arg':ref('Selector')})]}
D['CoverageProfile']=sealed('symphony.shv.coverage-profile.v1',{'as_of':date,'selector':ref('Selector')})
D['InspectInput']=obj({});D['CoverageDefaultInput']=obj({'as_of':date});D['CoveragePlanInput']=obj({'profile':ref('CoverageProfile'),'subjects':arr(ref('SubjectSummary'),128)})
D['CoverageResult']=sealed('symphony.shv.coverage-result.v1',{'profile':ref('CoverageProfile'),'subjects':arr(ref('SubjectSummary'),128),'decisions':arr(obj({'subject_id':id,'status':{'enum':['included','excluded','unresolved']}}),128),'counts':obj({x:{'type':'integer','minimum':0,'maximum':128} for x in ['included','excluded','unresolved']})})
D['CatalogueBuildInput']=obj({'source_root':txt(),'sources':arr(ref('Source'),8),'subjects':arr(ref('SubjectSpec'),32)})
D['Catalogue']=sealed('symphony.shv.catalogue.v1',{'sources':arr(ref('Source'),8),'subjects':arr(ref('Subject'),32),'mapping':arr(ref('SubjectSpec'),32)})
ids={**arr(id,128),'uniqueItems':True}
D['CatalogueQueryInput']=obj({'source_root':txt(),'catalogue':ref('Catalogue'),'subject_ids':ids})
D['QueryResult']=sealed('symphony.shv.query-result.v1',{'catalogue_digest':digest,'subject_ids':ids,'subjects':arr(ref('Subject'),32),'missing_subject_ids':ids})
D['ScalarValue']=copy.deepcopy(D['Value'])
D['Requirement']=obj({'id':id,'predicate':id,'operator':{'enum':['eq','contains','gte']},'value':ref('ScalarValue'),'qualifier':txt(256)})
D['EvaluateInput']=obj({'source_root':txt(),'catalogue':ref('Catalogue'),'subject_ids':ids,'requirements':arr(ref('Requirement'),32)})
D['Evaluation']=sealed('symphony.shv.evaluation.v1',{'catalogue_digest':digest,'subject_ids':ids,'requirements':arr(ref('Requirement'),32),'findings':arr(obj({'subject_id':id,'requirement_id':id,'status':{'enum':['supported','contradicted','unresolved']},'evidence':arr(ref('Assertion'),1)}),1024),'missing_subject_ids':ids})
G=json.loads((R/'modules/shv-graph-adapter/schemas/v1/graph-adapter.schema.json').read_bytes())['$defs']
for name in ['Graph','Node','Edge']:D[name]=copy.deepcopy(G[name])
D['GraphProjectInput']=obj({'source_root':txt(),'catalogue':ref('Catalogue')});D['GraphValidateInput']=obj({'source_root':txt(),'graph':ref('Graph')})
D['GraphValidation']=sealed('symphony.shv.graph-validation.v1',{'graph_digest':digest,'catalogue_digest':digest,'valid':{'const':True}})
# Generic descriptor definitions are framework-owned and shared verbatim.
for name in ['token','version','digest','featureId','operationId','operation','interaction','limits','Descriptor']:D[name]=copy.deepcopy(G[name])

# Versioned mapping addition; old untagged mappings remain structurally distinct.
D['QuarterValue']=obj({'precision':{'const':'quarter'},'source_text':{'type':'string','pattern':"^Q[1-4]'[0-9]{2}$"},'from':date,'through':date})
D['TableRowsValue']=obj({'columns':{**arr(txt(256),16),'minItems':1,'uniqueItems':True},'rows':{**arr({**arr({'type':'string','maxLength':4096},16),'minItems':1},32),'minItems':1}})
D['Value']['oneOf'] += [ref('QuarterValue'),ref('TableRowsValue')]
section=dict(txt(128),pattern=r'^[a-z][a-z0-9-]*#[^#]+$')
D['ScalarTableField']=obj({'predicate':id,'section':section,'label':txt(256),'next_label':{'anyOf':[txt(256),{'type':'null'}]},'value_type':{'enum':['string','integer','date','tokens','quarter_20yy']},'qualifier':txt(256)})
D['MatrixTableField']=obj({'predicate':id,'section':section,'columns':{**arr(txt(256),16),'minItems':1,'uniqueItems':True},'value_type':{'const':'table_rows'},'qualifier':txt(256)})
D['TableSubjectSpec']=obj({'id':id,'manufacturer':txt(256),'model':txt(256),'hardware_class':id,'source_id':id,'heading_section':section,'interpretation_profile':{'const':'scoped_tables.v1'},'fields':arr({'oneOf':[ref('ScalarTableField'),ref('MatrixTableField')]},16)})
D['LegacySubjectSpec']=D['SubjectSpec']
D['SubjectSpec']={'oneOf':[ref('LegacySubjectSpec'),ref('TableSubjectSpec')]}

schema={'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'symphony.shv.kernel-schema.v1','$comment':'Graph, Node, Edge copied verbatim from generic graph adapter contract. Structural schema checks do not replace source replay, canonical seals, sorting or runtime budgets.','$defs':D}
(B/'shv.schema.json').write_text(json.dumps(schema,indent=2)+'\n')
templates={'inspect':{},'coverage_default':{'as_of':None},'coverage_plan':{'profile':None,'subjects':[]},'catalogue_build':{'source_root':None,'sources':[],'subjects':[]},'catalogue_query':{'source_root':None,'catalogue':None,'subject_ids':[]},'evaluate':{'source_root':None,'catalogue':None,'subject_ids':[],'requirements':[]},'graph_project':{'source_root':None,'catalogue':None},'graph_validate':{'source_root':None,'graph':None}}
(B/'shv.templates.json').write_text(json.dumps(templates,indent=2)+'\n')
print('Wrote',len(D),'schema definitions and',len(templates),'unanswered templates')
