#!/usr/bin/env python3
"""Interrupt the actual connector translation unit; recover using installed qxctl.

Only the separately built test executable contains fault barriers. It is not an
installed adapter and does not attest its caller-supplied Installation value.
Every recovery uses the unchanged, exact receipt-owned production connector.
SIGKILL here is process-crash evidence, never simulated power-loss evidence.
"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import select
import signal
import subprocess
import time
import uuid
from installed_integration import Campaign, canonical, digest


class InterruptedCommitCampaign(Campaign):
    def __init__(self, args):
        super().__init__(args)
        self.faults = []
        self.cases = []
        self.summary.update({"recovery_scope": "SIGKILL at deterministic barriers in the connector's own prepare/commit implementation; production qxctl recovery.",
                             "fault_executable": str(args.fault_engine.resolve()),
                             "fault_executable_sha256": hashlib.sha256(args.fault_engine.read_bytes()).hexdigest(),
                             "fault_executable_installed": False,
                             "power_loss_tested": False})
        (self.out / "faults").mkdir(mode=0o700)
        self.save()

    def interrupt(self, name, stage, operation, payload):
        request = {"protocol": "symphony.knowledge.engine-process.v1", "request_id": str(uuid.uuid4()),
                   "correlation_id": str(uuid.uuid4()), "operation": operation,
                   "target_engine": "symphony-scv-graph-duckdb-connector",
                   "deadline_unix_ms": int(time.time() * 1000) + 30000, "payload": payload}
        prefix = self.out / "faults" / name
        prefix.with_suffix(".request.json").write_bytes(canonical(request))
        read_fd, write_fd = os.pipe()
        env = dict(os.environ, SYMPHONY_SCV_TEST_BARRIER=stage,
                   SYMPHONY_SCV_TEST_BARRIER_FD=str(write_fd))
        proc = None
        try:
            proc = subprocess.Popen([str(self.args.fault_engine.resolve())], cwd=self.root, env=env,
                                    pass_fds=(write_fd,), stdin=subprocess.PIPE,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            os.close(write_fd)
            write_fd = -1
            proc.stdin.write(canonical(request))
            proc.stdin.close()
            proc.stdin = None
            end = time.monotonic() + 20
            marker = b""
            while not marker.endswith(b"\n"):
                remaining = end - time.monotonic()
                assert remaining > 0 and select.select([read_fd], [], [], remaining)[0], (name, "barrier timeout")
                data = os.read(read_fd, 256)
                assert data, (name, "writer exited before barrier")
                marker += data
                assert len(marker) <= 128, "unexpected barrier framing"
            assert marker == (stage + "\n").encode(), (name, marker)
            # Observe an actual stopped process before delivering SIGKILL. No
            # exception unwinding, destructor rollback or graceful close occurs.
            while True:
                pid, state = os.waitpid(proc.pid, os.WUNTRACED | os.WNOHANG)
                if pid:
                    assert os.WIFSTOPPED(state) and os.WSTOPSIG(state) == signal.SIGSTOP, (name, state)
                    break
                assert time.monotonic() < end, (name, "writer did not stop")
                time.sleep(0.001)
            proc.kill()
            stdout, stderr = proc.communicate(timeout=5)
            prefix.with_suffix(".stdout").write_bytes(stdout)
            prefix.with_suffix(".stderr").write_bytes(stderr)
            self.check(proc.returncode == -signal.SIGKILL and not stdout,
                       name + ": stopped writer killed without a response or graceful cleanup")
            self.faults.append({"name": name, "stage": stage, "operation": operation,
                                "pid": proc.pid, "stop_signal": "SIGSTOP", "exit_code": proc.returncode,
                                "marker": marker.decode().strip(), "response_bytes": len(stdout),
                                "request": str(prefix.with_suffix('.request.json').relative_to(self.out)),
                                "writer_reaped": True})
            (self.out / "FAULTS.json").write_bytes(canonical(self.faults) + b"\n")
        finally:
            if proc is not None and proc.poll() is None:
                proc.kill()
                proc.communicate(timeout=5)
            os.close(read_fd)
            if write_fd >= 0:
                os.close(write_fd)

    def one_case(self, stage, shared=False):
        name = stage.replace('.', '-') + ('-shared' if shared else '')
        self.root = self.out / name
        self.root.mkdir(mode=0o700)
        self.namespace = name + '-sentinel'
        sentinel = self.qx(name + '-sentinel', 'import',
                           {'operation_id': 'sentinel', 'graph': self.graph, 'query_time': self.time})['connector_result']
        original = sentinel['intent']['snapshot']
        if not shared:
            self.namespace = name + '-target'
        payload = {'tops_id': self.tops, 'namespace': self.namespace, 'operation_id': 'target',
                   'graph': self.graph, 'query_time': self.time,
                   'owner': original['owner'], 'connector': original['connector']}
        snapshot = dict(original, namespace=self.namespace)
        snapshot.pop('digest')
        snapshot['digest'] = digest(snapshot)
        expected_intent = {'protocol': 'symphony.scv.graph-index-intent.v1', 'operation_id': 'target',
                           'snapshot': snapshot, 'validation_query_time': self.time}
        expected_intent['digest'] = digest(expected_intent)
        if stage.startswith('commit.'):
            prepared = self.native(name + '-prepare', 'prepare', payload, original['connector'])
            self.check(prepared['state'] == 'prepared' and prepared['intent'] == expected_intent,
                       name + ': exact intent durably prepared before faulted commit')
            fault_payload = {k: payload[k] for k in ('tops_id', 'namespace', 'operation_id')}
            fault_payload['expected_intent_digest'] = expected_intent['digest']
            operation = 'commit'
        else:
            fault_payload, operation = payload, 'prepare'
        self.interrupt(name, stage, operation, fault_payload)
        absent = stage == 'prepare.before_commit'
        committed = stage == 'commit.after_commit'
        observed = self.qx(name + '-status__target', 'status', ok=not absent)
        if not absent:
            status = observed['connector_result']
            self.check(status['state'] == ('committed' if committed else 'prepared') and
                       status['intent'] == expected_intent and status['index_verified'] == committed,
                       name + ': recovery observation matches the durable transaction boundary')
        visible = shared or committed
        export_input = {'snapshot_digest': snapshot['digest'], 'query_time': self.time}
        observed_export = self.qx(name + '-export-before', 'export', export_input, ok=visible)
        if visible:
            self.check(observed_export['connector_result']['snapshot'] == snapshot,
                       name + ': only the complete expected snapshot is visible before recovery')
        # An unrelated committed snapshot must survive every failure point.
        sentinel_export = self.qx(name + '-sentinel-after', 'export',
                                 {'snapshot_digest': original['digest'], 'query_time': self.time},
                                 namespace=original['namespace'])
        self.check(sentinel_export['connector_result']['snapshot'] == original,
                   name + ': earlier committed snapshot preserved in full')
        if absent:
            self.qx(name + '-missing-recover__target', 'recover', ok=False)
            recovered = self.qx(name + '-reimport', 'import',
                                {'operation_id': 'target', 'graph': self.graph, 'query_time': self.time})
        else:
            recovered = self.qx(name + '-recover__target', 'recover')
        result = recovered['connector_result']
        self.check(result['state'] == 'committed' and result['index_verified'] and
                   result['intent'] == expected_intent and
                   recovered['owner_evaluation']['graph_digest'] == self.graph['digest'],
                   name + ': installed qxctl recovers exact intent with original semantic owner')
        repeated = self.qx(name + '-repeat__target', 'recover')
        self.check(repeated == dict(recovered, operation='recover') if not absent else
                   repeated['connector_result'] == result and repeated['owner_evaluation'] == recovered['owner_evaluation'],
                   name + ': recovery retry preserves the complete logical result')
        # Full export validation reconstructs every expected row. A partial or
        # extra surviving inventory fails both native and independent Go checks.
        exported = self.qx(name + '-export-after', 'export', export_input)
        self.check(exported['connector_result']['snapshot'] == snapshot and
                   exported['connector_result']['counts'] == sentinel['counts'] and
                   exported['connector_result']['projection_digest'] == sentinel['projection_digest'],
                   name + ': complete row inventory and graph identity preserved after recovery')
        # Expected-state conflict still refuses after the writer disappeared.
        self.native(name + '-stale-commit', 'commit',
                    {'tops_id': self.tops, 'namespace': self.namespace, 'operation_id': 'target',
                     'expected_intent_digest': 'sha256:' + '0' * 64}, original['connector'], ok=False)
        self.cases.append({'name': name, 'stage': stage, 'shared_snapshot': shared,
                           'observed_state': 'absent' if absent else ('committed' if committed else 'prepared'),
                           'recovered_snapshot_digest': snapshot['digest'], 'status': 'passed'})
        (self.out / 'CASES.json').write_bytes(canonical(self.cases) + b'\n')

    def run(self):
        for stage in ('prepare.before_commit', 'prepare.after_commit', 'commit.after_snapshot',
                      'commit.after_row', 'commit.before_commit', 'commit.after_commit'):
            self.one_case(stage)
        for stage in ('commit.before_commit', 'commit.after_commit'):
            self.one_case(stage, shared=True)
        self.summary.update({'status': 'passed', 'interruption_cases': len(self.cases),
                             'faulted_writers_reaped': len(self.faults)})
        self.save()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('qxctl', 'connector-prefix', 'owner-prefix', 'graph', 'out', 'fault-engine'):
        p.add_argument('--' + name, type=Path, required=True)
    campaign = InterruptedCommitCampaign(p.parse_args())
    try:
        campaign.run()
    except Exception:
        campaign.summary['status'] = 'failed'
        campaign.save()
        raise
    print(json.dumps({'status': 'passed', 'cases': len(campaign.cases),
                      'calls': len(campaign.calls), 'assertions': len(campaign.assertions)}))


if __name__ == '__main__':
    main()
