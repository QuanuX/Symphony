"""Real installed structural store with caller-owned temporary data and exact writer receipts."""
import copy
from pathlib import Path
import subprocess
import sys
import tempfile
sys.path.insert(0, str(Path(__file__).resolve().parents[3] / 'tools/qxctl/tests/shv-invariants'))
from installed_support import Evidence, arguments, seal


def test_installed_store_provenance(args):
    evidence = Evidence(args)
    prefix = Path(args.prefix).resolve()
    receipt = prefix / 'share/symphony/receipts/shv-graph-duckdb-connector' / args.version / 'install-receipt.json'
    installation = evidence.installation('shv-graph-duckdb-connector', 'symphony-shv-graph-duckdb-connector',
                                         ['graph', 'store'], receipt)
    with tempfile.TemporaryDirectory(prefix='shv-store-installed-') as directory:
        root = Path(directory).resolve()
        root.chmod(0o700)
        def call(operation, payload, error_code=None):
            success = error_code is None
            completed = subprocess.run([installation['ExecutablePath']], cwd=root,
                                       input=evidence.request(installation['EngineID'], operation, payload),
                                       capture_output=True, timeout=40, env={})
            assert (completed.returncode == 0) == success, (completed.stdout, completed.stderr)
            return evidence.response(completed, installation, operation, payload, success, error_code)
        artifact = seal({'protocol': 'caller.example.v1', 'future_field': {'retired': None}})
        graph = seal({'protocol': 'symphony.graph.exchange.v1',
                      'owner': {'engine_id': 'caller-owner', 'engine_version': '1',
                                'artifact_protocol': artifact['protocol'], 'artifact_digest': artifact['digest']},
                      'owner_artifact': artifact,
                      'nodes': [{'id': 'a', 'labels': ['caller'], 'properties': {'custom': 16}},
                                {'id': 'b', 'labels': ['caller'], 'properties': {'retired': None}}],
                      'edges': [{'id': 'link', 'from': 'a', 'to': 'b', 'label': 'caller-link', 'properties': {}}]})
        scope = {'tops_id': '00000000-0000-4000-8000-000000000001', 'namespace': 'caller'}
        key = dict(scope, operation_id='one')
        payload = dict(key, graph=graph, connector=installation)
        prepared = call('prepare', payload)
        assert prepared['state'] == 'prepared'
        call('commit', dict(key, expected_intent_digest='sha256:' + '0' * 64), 'connector.invalid')
        committed = call('commit', dict(key, expected_intent_digest=prepared['intent']['digest']))
        assert committed['state'] == 'committed'
        exported = call('export', dict(scope, snapshot_digest=committed['snapshot_digest']))
        assert exported['snapshot']['graph'] == graph
        assert exported['snapshot']['connector'] == installation
        call('export', dict(scope, namespace='other', snapshot_digest=committed['snapshot_digest']), 'connector.invalid')
        forged = copy.deepcopy(graph)
        forged['edges'][0]['to'] = 'absent'
        call('prepare', dict(payload, operation_id='forged', graph=seal(forged)), 'graph.invalid_request')
        assert call('status', key) == committed
    evidence.finish('test_installed_store_provenance')


if __name__ == '__main__':
    test_installed_store_provenance(arguments())
