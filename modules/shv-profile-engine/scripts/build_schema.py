"""Receipt-owned profile engine resources; existing exact kernel schemas retained."""
from pathlib import Path
import copy,json
R=Path(__file__).resolve().parents[3];B=R/'modules/shv-profile-engine/schemas/v1'
def obj(p):return {'type':'object','properties':p,'required':list(p),'additionalProperties':False}
def arr(v,n):return {'type':'array','items':v,'maxItems':n}
def ref(s):return {'$ref':'#/$defs/'+s}
def txt(n=4096):return {'type':'string','minLength':1,'maxLength':n}
id={'type':'string','pattern':'^[A-Za-z0-9._-]{1,128}$'};digest={'type':'string','pattern':'^sha256:[0-9a-f]{64}$'}
def sealed(proto,fields):return obj({'protocol':{'const':proto},**fields,'digest':digest})
D=json.loads((R/'modules/shv-engine/schemas/v1/shv.schema.json').read_text())['$defs']
D['Metric']=obj({'predicate':id,'value_type':{'enum':['string','integer','date','tokens','quarter_20yy','table_rows']},'qualifier':txt(256),'required':{'type':'boolean'},'description':txt(),'extensions':{'type':'object'}})
D['ProfileCompileInput']=obj({'id':id,'revision':id,'hardware_class':id,'metrics':arr(ref('Metric'),16),'extensions':{'type':'object'}})
D['ClassProfile']=sealed('symphony.shv.class-profile.v1',{'definition':ref('ProfileCompileInput')})
# Diagnosis accepts declarations, before exact reader validation. In particular,
# a selector is text here; binding still uses the original kernel selector grammar.
for base in ['LegacySubjectSpec','TableSubjectSpec','PDFSubjectSpec','ScalarTableField','MatrixTableField']:
 D['Declaration'+base]=copy.deepcopy(D[base]);p=D['Declaration'+base]['properties']
 for key in ['heading_section','field_section','section']:
  if key in p and 'pattern' in p[key]:p[key]=txt(128)
D['DeclarationTableSubjectSpec']['properties']['fields']=arr({'oneOf':[ref('DeclarationScalarTableField'),ref('DeclarationMatrixTableField')]},16)
D['DeclarationPDFSubjectSpec']['properties']['document']={'oneOf':[obj({'decoder_root':txt(),'extraction':{'type':'object'}}),obj({'decoder_binding':id,'extraction':{'type':'object'}})]}
D['Declaration']={'oneOf':[ref('Declaration'+s)for s in ['LegacySubjectSpec','TableSubjectSpec','PDFSubjectSpec']]}
D['PortablePDFSubjectSpec']=copy.deepcopy(D['DeclarationPDFSubjectSpec']);D['PortablePDFSubjectSpec']['properties']['document']=obj({'decoder_binding':id,'extraction':{'type':'object'}})
D['PortableMapping']={'oneOf':[ref('DeclarationLegacySubjectSpec'),ref('DeclarationTableSubjectSpec'),ref('PortablePDFSubjectSpec')]}
D['MappingDiagnoseInput']=obj({'profile':ref('ClassProfile'),'mapping':arr(ref('Declaration'),32)})
D['Finding']=obj({'predicate':id,'status':{'enum':['matched','mismatch','unmapped_required','unmapped_optional']},'expected':ref('Metric'),'observed':{'anyOf':[{'type':'null'},obj({'value_type':txt(32),'qualifier':txt(256)})]},'differences':{**arr({'enum':['value_type','qualifier']},2),'uniqueItems':True}})
D['MappingDiagnostics']=sealed('symphony.shv.mapping-diagnostics.v1',{'input':ref('MappingDiagnoseInput'),'subjects':arr(obj({'subject_id':id,'status':{'enum':['conformant','incomplete','not_applicable']},'findings':arr(ref('Finding'),16),'extension_predicates':arr(id,16)}),32),'counts':obj({x:{'type':'integer','minimum':0,'maximum':32} for x in ['conformant','incomplete','not_applicable']}),'evidence_scope':{'const':'mapping_declarations_only'}})
D['UniverseBuildInput']=obj({'id':id,'revision':id,'kernel_version':{'const':'0.3.0-dev'},'coverage':ref('CoverageProfile'),'profiles':arr(ref('ClassProfile'),16),'sources':arr(ref('Source'),8),'mapping':arr(ref('PortableMapping'),32),'locators':arr(obj({'source_id':id,'uri':txt(),'upstream_revision':{'anyOf':[{'type':'null'},txt(256)]}}),8),'extensions':{'type':'object'}})
D['Universe']=sealed('symphony.shv.universe.v1',{'definition':ref('UniverseBuildInput')})
D['UniverseBindInput']=obj({'universe':ref('Universe'),'bindings':obj({'source_root':txt(),'decoders':{'type':'object','propertyNames':id,'additionalProperties':txt()}})})
D['UniverseBinding']=sealed('symphony.shv.universe-binding.v1',{'input':ref('UniverseBindInput'),'reader':obj({'engine_id':{'const':'symphony-shv'},'version':{'const':'0.3.0-dev'},'mode':{'const':'compiled_exact_contract'}}),'catalogue_input':ref('CatalogueBuildInput'),'catalogue':ref('Catalogue'),'coverage':ref('CoverageResult'),'conformance':arr(ref('MappingDiagnostics'),16),'unprofiled_classes':{**arr(id,32),'uniqueItems':True},'canonical_apply_enabled':{'const':False}})
# Ship only the reachable contract definitions, preserving their exact shapes.
needed=set()
def visit(name):
 if name in needed:return
 needed.add(name)
 def walk(v):
  if isinstance(v,dict):
   if '$ref' in v and v['$ref'].startswith('#/$defs/'):visit(v['$ref'].split('/')[-1])
   for child in v.values():walk(child)
  elif isinstance(v,list):
   for child in v:walk(child)
 walk(D[name])
for name in ['InspectInput','Descriptor','ProfileCompileInput','ClassProfile','MappingDiagnoseInput','MappingDiagnostics','UniverseBuildInput','Universe','UniverseBindInput','UniverseBinding']:visit(name)
D={k:v for k,v in D.items() if k in needed}
schema={'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'symphony.shv.profile-schema.v1','$comment':'Structural checks only. Runtime enforces seals, correspondence, bounds, unique identities and source replay. Declaration checks do not interpret selectors or PDF extraction objects. The compiled exact kernel validates them at bind.','$defs':D}
(B/'profile.schema.json').write_text(json.dumps(schema,indent=2)+'\n')
templates={'profile_compile':{'id':None,'revision':None,'hardware_class':None,'metrics':[],'extensions':{}},'mapping_diagnose':{'profile':None,'mapping':[]},'universe_build':{'id':None,'revision':None,'kernel_version':'0.3.0-dev','coverage':None,'profiles':[],'sources':[],'mapping':[],'locators':[],'extensions':{}},'universe_bind':{'universe':None,'bindings':{'source_root':None,'decoders':{}}}}
(B/'profile.templates.json').write_text(json.dumps(templates,indent=2)+'\n')
