"""Real installed pure publication owner; empty caller catalogue is not live head-write authority."""
import copy
from pathlib import Path
import subprocess
import sys
import tempfile
sys.path.insert(0, str(Path(__file__).resolve().parents[3] / 'tools/qxctl/tests/shv-invariants'))
from installed_support import Evidence, arguments, seal


def test_installed_publication_provenance(args):
    evidence = Evidence(args)
    prefix = Path(args.prefix).resolve()
    receipt = prefix / 'share/symphony/receipts/shv-publication-engine' / args.version / 'install-receipt.json'
    installation = evidence.installation('shv-publication-engine', 'symphony-shv-publication',
                                         ['catalogue', 'publication'], receipt)
    partition_prefix = Path(args.partition_prefix).resolve()
    partition_receipt = partition_prefix / 'share/symphony/receipts/shv-partition-engine' / args.partition_version / 'install-receipt.json'
    partition = evidence.installation('shv-partition-engine', 'symphony-shv-partition', ['partition'],
                                     partition_receipt, partition_prefix, args.partition_version)
    with tempfile.TemporaryDirectory(prefix='shv-publication-installed-') as directory:
        def call(operation, payload, error_code=None):
            success = error_code is None
            completed = subprocess.run([installation['ExecutablePath']], cwd=directory,
                                       input=evidence.request(installation['EngineID'], operation, payload),
                                       capture_output=True, timeout=40, env={})
            assert (completed.returncode == 0) == success, (completed.stdout, completed.stderr)
            return evidence.response(completed, installation, operation, payload, success, error_code)
        manifest = seal({'protocol': 'symphony.shv.partition-manifest.v1', 'entries': [],
                         'required_references': [], 'loaded_count': 0, 'missing_count': 0,
                         'complete_inventory': True, 'reference_statuses': []})
        desired = {'catalogue_id': 'caller', 'tops_id': '00000000-0000-4000-8000-000000000001',
                   'manifest': manifest, 'policy': {'missing_partitions': 'reject', 'missing_references': 'reject'},
                   'partition_installation': partition, 'members': []}
        payload = {'operation_id': 'first', 'current': None, 'desired': desired, 'reason': 'caller test'}
        plan = call('publication_plan', payload)
        assert plan['head']['definition'] == desired
        head = call('publication_reduce', {'current': None, 'plan': plan})['head']
        assert head['generation'] == 1 and head['definition']['partition_installation'] == partition
        assert call('publication_status', {'history': [head]})['head'] == head
        forged = copy.deepcopy(plan)
        forged['head']['generation'] = 2
        forged['head'] = seal(forged['head'])
        call('publication_reduce', {'current': None, 'plan': seal(forged)}, 'shv-publication.invalid')
        call('publication_status', {'history': [head, head]}, 'shv-publication.invalid')
        substituted = copy.deepcopy(desired)
        substituted['partition_installation']['Version'] = 'latest'
        call('publication_plan', dict(payload, desired=substituted), 'shv-publication.invalid')
    evidence.finish('test_installed_publication_provenance')


def extra_arguments(parser):
    parser.add_argument('--partition-prefix', required=True)
    parser.add_argument('--partition-version', required=True)


if __name__ == '__main__':
    test_installed_publication_provenance(arguments(extra_arguments))
