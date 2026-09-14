"""Generate exact CLI composition definitions from unchanged native owner schemas."""
import json,copy
from pathlib import Path
r=Path(__file__).resolve().parents[3]
def scoped(path,name):
 s=json.loads((r/path).read_text());s.pop('$id',None)
 def fix(v):
  if isinstance(v,dict):return {k:('#/$defs/'+name+'/$defs/'+x[len('#/$defs/'):] if k=='$ref' and x.startswith('#/$defs/') else fix(x)) for k,x in v.items()}
  if isinstance(v,list):return [fix(x) for x in v]
  return v
 return fix(s)
def ref(n):return {'$ref':'#/$defs/'+n}
def obj(p):return {'type':'object','additionalProperties':False,'required':list(p),'properties':p}
d={'Kernel':scoped('modules/shv-engine/schemas/v1/shv.schema.json','Kernel'),'SourceOwner':scoped('modules/shv-source-engine/schemas/v1/source.schema.json','SourceOwner')}
a=json.loads((r/'tools/qxctl/cmd/qxctl/shv_activation.schema.json').read_text());d['Installation']=a['$defs']['ActivationInstallation']
c=copy.deepcopy(d['SourceOwner']['$defs']['CaptureImportInput']);c['required']=[k for k in c['required'] if k not in ['source_root','source']]
for k in ['source_root','source']:c['properties'].pop(k)
d['CaptureSpec']=c
req=obj({'expected_source_digest':ref('SourceOwner/$defs/digest'),'captures':{'type':'array','minItems':1,'maxItems':8,'items':ref('CaptureSpec')},'mapping':{'type':'array','maxItems':32,'items':ref('Kernel/$defs/SubjectSpec')},'profile':ref('Kernel/$defs/CoverageProfile'),'subject_ids':{'type':'array','maxItems':32,'uniqueItems':True,'items':ref('Kernel/$defs/token')},'requirements':{'type':'array','maxItems':32,'items':ref('Kernel/$defs/Requirement')}});d['Request']=req
p={'protocol':{'const':'symphony.qxctl.shv-refresh-bundle.v1'},'tops_id':{'type':'string','format':'uuid'},'source_id':ref('SourceOwner/$defs/token'),'source':ref('SourceOwner/$defs/Source'),'source_installation':ref('Installation'),'kernel_installation':ref('Installation'),'request':ref('Request'),'captures':{'type':'array','minItems':1,'maxItems':8,'items':ref('SourceOwner/$defs/Capture')},'catalogue':ref('Kernel/$defs/Catalogue'),'coverage':ref('Kernel/$defs/CoverageResult'),'evaluation':ref('Kernel/$defs/Evaluation'),'source_graph':ref('SourceOwner/$defs/Graph'),'catalogue_graph':ref('Kernel/$defs/Graph'),'digest':ref('SourceOwner/$defs/digest')};d['Bundle']=obj(p)
d['Verification']=obj({'protocol':{'const':'symphony.qxctl.shv-refresh-verification.v1'},'bundle_digest':ref('SourceOwner/$defs/digest'),'selected_source_digest':ref('SourceOwner/$defs/digest'),'current_source_digest':ref('SourceOwner/$defs/digest'),'source_is_current':{'type':'boolean'},'valid':{'const':True},'digest':ref('SourceOwner/$defs/digest')})
base={'source_installation':ref('Installation'),'kernel_installation':ref('Installation'),'digest':ref('SourceOwner/$defs/digest')}
d['Schema']=obj({**base,'protocol':{'const':'symphony.qxctl.shv-refresh-schema.v1'},'origin':{'const':'qxctl_embedded'},'schema':{'type':'object'}})
d['Template']=obj({**base,'protocol':{'const':'symphony.qxctl.shv-refresh-template.v1'},'status':{'const':'unanswered_template_not_validated_input'},'template':obj({k:{'type':'null'} for k in req['properties']})})
s={'$schema':'https://json-schema.org/draft/2020-12/schema','$defs':d}
# Additive CLI comparison definitions. Exact owner schemas above remain unchanged.
endpoint=obj({k:{'type':'string','minLength':1} for k in ['bundle_path','source_root','state_root','tops_id','source_id','source_prefix','source_version','kernel_prefix','kernel_version']})
d['ComparisonEndpoint']=endpoint;d['ComparisonRequest']=obj({'previous':ref('ComparisonEndpoint'),'current':ref('ComparisonEndpoint')})
nullable=lambda value:{'anyOf':[value,{'type':'null'}]}
base={'kind':{'enum':['added','removed','changed']},'subject_id':ref('Kernel/$defs/token')}
d['SubjectChange']=obj({**base,'previous':nullable(ref('Kernel/$defs/SubjectSummary')),'current':nullable(ref('Kernel/$defs/SubjectSummary'))})
d['AssertionChange']=obj({**base,'predicate':ref('Kernel/$defs/token'),'previous':nullable(ref('Kernel/$defs/Assertion')),'current':nullable(ref('Kernel/$defs/Assertion'))})
names=['source','source_installation','kernel_installation','mapping','profile','subject_ids','requirements','capture_bodies','capture_manifests','capture_observations','catalogue','coverage','evaluation','source_graph','catalogue_graph']
d['Dimension']=obj({'dimension':{'enum':names},'changed':{'type':'boolean'},'previous_digest':ref('SourceOwner/$defs/digest'),'current_digest':ref('SourceOwner/$defs/digest')})
d['Comparison']=obj({'protocol':{'const':'symphony.qxctl.shv-refresh-comparison.v1'},'tops_id':{'type':'string','format':'uuid'},'source_id':ref('SourceOwner/$defs/token'),'previous_bundle_digest':ref('SourceOwner/$defs/digest'),'current_bundle_digest':ref('SourceOwner/$defs/digest'),'same_bundle':{'type':'boolean'},'previous_replay':ref('Verification'),'current_replay':ref('Verification'),'observation_scope':{'const':'sequential_source_replays'},'causal_attribution':{'const':'not_inferred'},'dimensions':{'type':'array','minItems':15,'maxItems':15,'items':ref('Dimension')},'subject_changes':{'type':'array','maxItems':64,'items':ref('SubjectChange')},'assertion_changes':{'type':'array','maxItems':512,'items':ref('AssertionChange')},'digest':ref('SourceOwner/$defs/digest')})
old=d['Template']['properties']['template'];blank=obj({k:{'type':'null'} for k in endpoint['properties']});d['Template']['properties']['template']={'oneOf':[old,obj({'previous':blank,'current':blank})]}
(r/'tools/qxctl/cmd/qxctl/shv_refresh.schema.json').write_text(json.dumps(s,indent=2)+'\n')
