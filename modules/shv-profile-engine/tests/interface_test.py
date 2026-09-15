"""Historical parity and adversarial authoring checks; no semantic oracle generation."""
from pathlib import Path
import argparse,copy,hashlib,importlib.util,json,subprocess,tempfile,unittest
M=Path(__file__).resolve().parents[1];R=M.parents[1]
spec=importlib.util.spec_from_file_location('generator',M/'tools/generate_interface.py');g=importlib.util.module_from_spec(spec);spec.loader.exec_module(g)
p=argparse.ArgumentParser();p.add_argument('--engine');args,rest=p.parse_known_args()
class InterfaceTests(unittest.TestCase):
 def test_frozen_native_descriptor(self):
  if not args.engine:self.skipTest('native executable not selected')
  actual=json.loads(subprocess.check_output([args.engine,'--descriptor']));expected=copy.deepcopy(g.read(M/'tests/fixtures/interface-history.v1.json')['descriptors']['0.2.0-dev']);expected['engine_version']='0.3.0-dev';expected.pop('descriptor_digest');expected['descriptor_digest']='sha256:'+hashlib.sha256(json.dumps(expected,sort_keys=True,separators=(',',':')).encode()).hexdigest();self.assertEqual(actual,expected)
 def test_deterministic_projection(self):
  d=g.validate(g.read(M/'OWNER-INTERFACE.json'));self.assertEqual(g.outputs(d),g.outputs(copy.deepcopy(d)))
  for path,data in g.outputs(d).items():self.assertEqual((R/path).read_text(),data)
 def test_rejects_declaration_mutations(self):
  original=g.read(M/'OWNER-INTERFACE.json')
  mutations={
   'extra_field':lambda d:d.update(extra=True),
   'missing_field':lambda d:d.pop('schemas'),
   'unknown_release':lambda d:d['releases'].update({'latest':[]}),
   'old_admission':lambda d:d['releases']['0.1.0-dev'].append('references_analyze'),
   'duplicate_operation':lambda d:d['operations'].append(d['operations'][0]),
   'new_mutation':lambda d:d['operations'][0].update(mutability='write'),
   'protocol_change':lambda d:d['operations'][0].update(output_protocol='invented'),
   'interaction_change':lambda d:d['operations'][0].update(administrative_interactions=['mutate']),
   'new_reader':lambda d:d.update(embedded_kernel_version='latest'),
   'escaping_companion':lambda d:d['companions'].append('../escape'),
   'duplicate_schema':lambda d:d['schemas'].append(d['schemas'][0]),
   'missing_schema':lambda d:d['schemas'].pop(),
  }
  for name,mutate in mutations.items():
   with self.subTest(name=name):
    d=copy.deepcopy(original);mutate(d)
    with self.assertRaises((AssertionError,KeyError)):g.validate(d)
 def test_rejects_duplicate_json_keys(self):
  with tempfile.TemporaryDirectory()as td:
   p=Path(td)/'duplicate.json';p.write_text('{"module_id":"first","module_id":"second"}')
   with self.assertRaises(ValueError):g.read(p)
 def test_rejects_schema_escape_and_missing_target(self):
  import shutil
  for ref in ['../../outside.json','#/$defs/Absent']:
   with self.subTest(ref=ref),tempfile.TemporaryDirectory()as td:
    root=Path(td)
    for path in [g.MODULE+'/tests/fixtures/interface-history.v1.json',*g.read(M/'OWNER-INTERFACE.json')['schemas'],*g.read(M/'OWNER-INTERFACE.json')['companions']]:
     dst=root/path;dst.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(R/path,dst)
    path=root/g.MODULE/'schemas/v1/profile.schema.json';doc=g.read(path);doc['$ref']=ref;path.write_text(json.dumps(doc))
    with self.assertRaises((AssertionError,KeyError)):g.validate(g.read(M/'OWNER-INTERFACE.json'),root)
if __name__=='__main__':unittest.main(argv=['interface_test',*rest])
