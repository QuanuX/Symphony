"""Real installed PDF source-replay boundary; requires a caller-selected retained PDF request."""
import copy
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import json
sys.path.insert(0, str(Path(__file__).resolve().parents[3] / 'tools/qxctl/tests/shv-invariants'))
from installed_support import Evidence, arguments, seal


def test_installed_pdf_source_replay(args):
    evidence = Evidence(args)
    prefix = Path(args.prefix).resolve()
    receipt = prefix / 'share/symphony/receipts/shv-pdf-adapter' / args.version / 'install-receipt.json'
    installation = evidence.installation('shv-pdf-adapter', 'symphony-shv-pdf', ['pdf'], receipt)
    def call(operation, payload, error_code=None):
        success = error_code is None
        completed = subprocess.run([installation['ExecutablePath']],
                                   input=evidence.request(installation['EngineID'], operation, payload),
                                   capture_output=True, timeout=40, env={})
        assert (completed.returncode == 0) == success, (completed.stdout, completed.stderr)
        return evidence.response(completed, installation, operation, payload, success, error_code)
    request = json.loads(Path(args.request).read_bytes())
    with tempfile.TemporaryDirectory(prefix='shv-pdf-installed-') as directory:
        root = Path(directory).resolve()
        original = Path(request['source_root']) / request['source']['path']
        local = root / 'original.pdf'
        shutil.copyfile(original, local)
        request = copy.deepcopy(request)
        request['source_root'] = str(root)
        request['source']['path'] = local.name
        extracted = call('extract', request)
        assert len(extracted['rows']) == 28 and extracted['documentary_lineages'] == 1
        assert extracted['namespace'] == 'AMD.OPN' and extracted['namespace_equivalence'] == 'not_asserted'
        graph = call('graph_project', request)
        assert len(graph['nodes']) == 30 and len(graph['edges']) == 57
        call('graph_validate', {'request': request, 'graph': graph})
        forged = copy.deepcopy(graph)
        forged['edges'][0]['properties']['qualifier'] = 'issuer=AMD;namespace=product-id-tray;profile=1'
        call('graph_validate', {'request': request, 'graph': seal(forged)}, 'shv-pdf.invalid')
        wrong_decoder = copy.deepcopy(request)
        wrong_decoder['decoder']['digest'] = 'sha256:' + '0' * 64
        call('extract', wrong_decoder, 'shv-pdf.invalid')
        local.write_bytes(b'changed retained original')
        call('graph_validate', {'request': request, 'graph': graph}, 'shv-pdf.invalid')
    evidence.finish('test_installed_pdf_source_replay')


if __name__ == '__main__':
    test_installed_pdf_source_replay(arguments(lambda parser: parser.add_argument('--request', required=True)))
