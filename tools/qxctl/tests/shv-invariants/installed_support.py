"""Shared evidence mechanics for explicit, receipt-backed SHV process tests.

This test helper verifies existing selected packages; it neither installs nor
modifies them. Native owner wrappers supply independently asserted semantic cases.
"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import time


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()


def digest(data):
    return 'sha256:' + hashlib.sha256(data).hexdigest()


def seal(value, field='digest'):
    value = copy.deepcopy(value)
    value.pop(field, None)
    value[field] = digest(canonical(value))
    return value


def arguments(extra=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--qxctl', required=True)
    parser.add_argument('--prefix', required=True)
    parser.add_argument('--version', required=True)
    parser.add_argument('--output', required=True)
    if extra:
        extra(parser)
    return parser.parse_args()


class Evidence:
    def __init__(self, args):
        self.args = args
        self.calls = []
        self.installations = []
        self.verified_receipts = []
        self.qxctl_inspections = []
        self.directory = Path(args.output).resolve()
        self.directory.mkdir(parents=True, exist_ok=True)

    def installation(self, module, engine, family, receipt_path, prefix=None, version=None):
        prefix = Path(prefix or self.args.prefix).resolve()
        version = version or self.args.version
        receipt_path = Path(receipt_path)
        assert receipt_path.is_file() and not receipt_path.is_symlink(), receipt_path
        # qxctl owns complete no-follow receipt, package and compiled-admission
        # verification. A made-up JSON Installation is never accepted here.
        prefix_flag, version_flag = ('--connector-prefix', '--connector-version') if module == 'shv-graph-duckdb-connector' else ('--prefix', '--version')
        checked = subprocess.run([self.args.qxctl, 'shv', *family, 'inspect', prefix_flag,
                                  str(prefix), version_flag, version, '--json'],
                                 capture_output=True, timeout=40)
        assert checked.returncode == 0, (checked.stdout, checked.stderr)
        assert not checked.stderr, checked.stderr
        envelope = json.loads(checked.stdout)
        if 'result' in envelope:
            assert envelope['outcome'] == 'ok' and envelope == seal(envelope, 'response_digest')
            descriptor = envelope['result']
        else:
            descriptor = envelope
        assert descriptor == seal(descriptor, 'descriptor_digest')
        assert descriptor['engine_id'] == engine and descriptor['engine_version'] == version
        receipt = json.loads(receipt_path.read_bytes())
        assert receipt == seal(receipt, 'receipt_digest')
        assert receipt['module_id'] == module and receipt['version'] == version
        executable = prefix / 'libexec' / 'symphony' / module / version / engine
        owned = [item for item in receipt['files'] if item['path'] == str(executable.relative_to(prefix))]
        assert len(owned) == 1 and owned[0]['digest'] == digest(executable.read_bytes())
        installation = dict(Role=module, ModuleID=module, EngineID=engine, Version=version,
                            Prefix=str(prefix), ReceiptPath=str(receipt_path),
                            ReceiptProtocol=receipt['protocol'], ReceiptDigest=receipt['receipt_digest'],
                            ExecutablePath=str(executable), ExecutableDigest=owned[0]['digest'])
        self.installations.append(installation)
        self.verified_receipts.append({"path": str(receipt_path), "receipt": receipt})
        self.qxctl_inspections.append({"module": module, "response": envelope})
        return installation

    def request(self, engine, operation, payload):
        return canonical(dict(protocol='symphony.knowledge.engine-process.v1',
                              request_id='installed-invariant', correlation_id='installed-invariant',
                              target_engine=engine, operation=operation,
                              deadline_unix_ms=int(time.time() * 1000) + 30000, payload=payload))

    def response(self, completed, installation, operation, payload, success, error_code=None):
        assert (completed.returncode == 0) == success, (operation, completed.stdout, completed.stderr)
        assert not completed.stderr, completed.stderr
        value = json.loads(completed.stdout)
        assert set(value) == {'protocol', 'request_id', 'correlation_id', 'operation', 'engine_id',
                              'engine_version', 'outcome', 'result', 'error', 'response_digest'}
        assert value['protocol'] == 'symphony.knowledge.engine-process.v1'
        assert value['request_id'] == 'installed-invariant' and value['correlation_id'] == 'installed-invariant'
        assert value == seal(value, 'response_digest'), operation
        assert value['engine_id'] == installation['EngineID']
        assert value['engine_version'] == installation['Version']
        assert value['operation'] == operation
        assert value['outcome'] == ('ok' if success else 'error')
        if not success:
            assert isinstance(error_code, str) and value['error']['code'] == error_code, value
        else:
            assert error_code is None and value['error'] is None
        self.calls.append(dict(operation=operation, expected_success=success, expected_error_code=error_code,
                               exit_code=completed.returncode, payload=payload, response=value))
        return value.get('result')

    def finish(self, case):
        result = dict(status='passed', case=case, installations=self.installations,
                      verified_receipts=self.verified_receipts, qxctl_inspections=self.qxctl_inspections,
                      arguments=vars(self.args), qxctl_digest=digest(Path(self.args.qxctl).read_bytes()),
                      calls=self.calls, call_count=len(self.calls),
                      expected_rejections=sum(not call['expected_success'] for call in self.calls),
                      installs_or_mutates_packages=False)
        (self.directory / 'ACCEPTANCE.json').write_text(json.dumps(result, indent=2, sort_keys=True) + '\n')
        print(json.dumps({key: result[key] for key in ['status', 'case', 'call_count', 'expected_rejections']}))
