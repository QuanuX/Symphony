"""Real installed profile binding over a reproducible synthetic source; no hardware fact claim."""
from pathlib import Path
import subprocess
import sys
import tempfile
sys.path.insert(0, str(Path(__file__).resolve().parents[3] / 'tools/qxctl/tests/shv-invariants'))
from installed_support import Evidence, arguments, digest, seal


def test_installed_profile_source_binding(args):
    evidence = Evidence(args)
    prefix = Path(args.prefix).resolve()
    receipt = prefix / 'share/symphony/receipts/shv-profile-engine' / args.version / 'install-receipt.json'
    installation = evidence.installation('shv-profile-engine', 'symphony-shv-profile', ['profile'], receipt)
    def call(operation, payload, error_code=None):
        success = error_code is None
        completed = subprocess.run([installation['ExecutablePath']],
                                   input=evidence.request(installation['EngineID'], operation, payload),
                                   capture_output=True, timeout=40, env={})
        assert (completed.returncode == 0) == success, (completed.stdout, completed.stderr)
        return evidence.response(completed, installation, operation, payload, success, error_code)
    with tempfile.TemporaryDirectory(prefix='shv-profile-installed-') as directory:
        root = Path(directory).resolve()
        source = root / 'source.html'
        content = b'<div id="overview"><h1>Synthetic GPU</h1></div><article id="spec"><dl><dt>Value</dt><dd>16</dd><dt>End</dt><dd>done</dd></dl></article>'
        source.write_bytes(content)
        mapping = {'id': 'gpu', 'manufacturer': 'Synthetic fixture', 'model': 'Synthetic GPU',
                   'hardware_class': 'gpu', 'source_id': 'source', 'heading_section': 'div#overview',
                   'field_section': 'article#spec', 'fields': [{'predicate': 'memory', 'value_type': 'integer',
                   'qualifier': 'unit=caller', 'label': 'Value', 'next_label': 'End'}]}
        profile = call('profile_compile', {'id': 'caller', 'revision': 'v1', 'hardware_class': 'gpu',
                   'metrics': [{'predicate': 'memory', 'value_type': 'integer', 'qualifier': 'unit=caller',
                   'required': True, 'description': 'Synthetic caller metric', 'extensions': {}}], 'extensions': {}})
        definition = {'id': 'selected', 'revision': 'v1', 'kernel_version': '0.3.0-dev',
                      'coverage': seal({'protocol': 'symphony.shv.coverage-profile.v1', 'as_of': '2026-09-15',
                                        'selector': {'op': 'all'}}), 'profiles': [profile],
                      'sources': [{'id': 'source', 'path': source.name, 'bytes': len(content),
                                   'digest': digest(content), 'format': 'html'}],
                      'mapping': [mapping], 'locators': [], 'extensions': {}}
        universe = call('universe_build', definition)
        payload = {'universe': universe, 'bindings': {'source_root': str(root), 'decoders': {}}}
        bound = call('universe_bind', payload)
        assert bound['catalogue']['subjects'][0]['assertions'][0]['value'] == 16
        assert bound['coverage']['counts'] == {'included': 1, 'excluded': 0, 'unresolved': 0}
        assert bound['reader']['version'] == '0.3.0-dev' and not bound['canonical_apply_enabled']
        source.write_bytes(content.replace(b'>16<', b'>32<'))
        call('universe_bind', payload, 'shv.invalid')
        source.unlink()
        target = root / 'other.html'
        target.write_bytes(content)
        source.symlink_to(target)
        call('universe_bind', payload, 'path.file_unreadable')
        invalid = dict(definition, kernel_version='latest')
        call('universe_build', invalid, 'shv.invalid')
    evidence.finish('test_installed_profile_source_binding')


if __name__ == '__main__':
    test_installed_profile_source_binding(arguments())
