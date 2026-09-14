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
(r/'tools/qxctl/cmd/qxctl/shv_refresh.schema.json').write_text(json.dumps(s,indent=2)+'\n')
