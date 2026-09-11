#!/usr/bin/env python3
"""Exercise a receipt-owned standalone engine through its real process boundary."""
import argparse
import hashlib
import json
import pathlib
import subprocess
import tempfile
import time
import unittest

parser = argparse.ArgumentParser()
parser.add_argument('--build', required=True)
parser.add_argument('--domain', required=True)
parser.add_argument('--version', required=True)
ARGS, REST = parser.parse_known_args()


def digest(value):
    return 'sha256:' + hashlib.sha256(json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


class InstalledProcessTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory(prefix='scv-installed-')
        cls.prefix = pathlib.Path(cls.temp.name)
        subprocess.run(['cmake', '--install', ARGS.build, '--prefix', str(cls.prefix)], check=True, capture_output=True)
        cls.module = ARGS.domain + '-engine'
        cls.engine_id = 'symphony-' + ARGS.domain
        cls.binary = cls.prefix / 'libexec/symphony' / cls.module / ARGS.version / cls.engine_id
        cls.receipt = json.loads((cls.prefix / 'share/symphony/receipts' / cls.module / ARGS.version / 'install-receipt.json').read_text())
        if ARGS.domain.startswith('schv-'):
            cls.family, cls.provider = 'schv', ARGS.domain[5:]
        elif ARGS.domain in ('scev', 'scev-cf'):
            cls.family, cls.provider = 'scev', 'cf'
        else:
            cls.family, cls.provider = 'schv', 'fixture'

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def request(self, operation, payload, **overrides):
        result = {'protocol': 'symphony.knowledge.engine-process.v1', 'request_id': 'fixture-request',
                  'correlation_id': 'fixture-correlation', 'operation': operation, 'target_engine': self.engine_id,
                  'deadline_unix_ms': int(time.time()*1000)+5000, 'payload': payload}
        result.update(overrides)
        return result

    def invoke(self, operation, payload, **overrides):
        raw = json.dumps(self.request(operation, payload, **overrides))
        proc = subprocess.run([str(self.binary)], input=raw, text=True, capture_output=True, timeout=6, env={})
        response = json.loads(proc.stdout)
        self.assertEqual(proc.stdout.count('\n'), 1)
        stored = response.pop('response_digest')
        self.assertEqual(stored, digest(response))
        response['response_digest'] = stored
        return proc, response

    def desired(self):
        return {'source_id': 'fixture-docs', 'provider_id': self.provider, 'family_id': self.family,
                'publisher': 'Synthetic publisher', 'authority_role': 'user_declared', 'scope': 'Synthetic documentation',
                'locators': [{'locator_id': 'docs', 'uri': 'https://example.invalid/docs.md', 'role': 'preferred',
                              'format': 'markdown', 'selector': 'fixture-a'}], 'continuity_evidence': ['fixture']}

    def initial(self):
        proc, response = self.invoke('source_plan', {'operation_id': 'onboard', 'current': None, 'desired': self.desired(), 'reason': 'Fixture onboarding'})
        self.assertEqual(proc.returncode, 0, response)
        return response['result']['source']

    def test_receipt_owned_standalone_process(self):
        self.assertEqual(self.receipt['component_id'], self.module)
        self.assertEqual(self.receipt['engine_id'], self.engine_id)
        self.assertEqual(self.receipt['version'], ARGS.version)
        proc, response = self.invoke('inspect', {})
        self.assertEqual(proc.returncode, 0, response)
        self.assertEqual(response['engine_id'], self.engine_id)
        self.assertEqual(response['request_id'], 'fixture-request')
        self.assertEqual(response['result']['engine_version'], ARGS.version)
        self.assertFalse(response['result']['canonical_apply_enabled'])
        self.assertEqual(len(response['result']['operations']), {'0.1.0-dev': 13, '0.2.0-dev': 17, '0.3.0-dev': 20, '0.4.0-dev': 21, '0.5.0-dev': 22, '0.6.0-dev':26, '0.7.0-dev':26}[ARGS.version])

    def test_profile_preparation_and_packaged_discovery_evidence(self):
        if ARGS.version not in ('0.4.0-dev', '0.5.0-dev', '0.6.0-dev', '0.7.0-dev'):
            self.skipTest('profile preparation is an exact .4 addition')
        draft = {'protocol': 'symphony.scv.interpretation-profile.v1', 'profile_id': 'fixture-profile',
                 'profile_version': 'test-1', 'provider_id': self.provider, 'source_id': 'fixture-docs',
                 'locator_id': 'docs', 'media_types': ['text/markdown'], 'authored_by': 'test fixture',
                 'rationale': 'Empty authored mapping tests structure, asserts no provider facts', 'rules': []}
        proc, response = self.invoke('profile_prepare', {'profile': draft})
        self.assertEqual(proc.returncode, 0, response)
        self.assertEqual(response['result'], {**draft, 'digest': digest(draft)})
        proc, response = self.invoke('profile_prepare', {'profile': {**draft, 'digest': digest(draft)}})
        self.assertNotEqual(proc.returncode, 0)
        catalog_path = f'share/symphony/schemas/{self.module}/{ARGS.version}/schema-catalog.json'
        owned = {item['path']: item for item in self.receipt['files']}
        self.assertIn(catalog_path, owned)
        raw = (self.prefix / catalog_path).read_bytes()
        self.assertEqual(owned[catalog_path]['digest'], 'sha256:' + hashlib.sha256(raw).hexdigest())
        catalog = json.loads(raw)
        self.assertEqual(catalog['engine_version'], ARGS.version)
        self.assertTrue(any(e['protocol'] == 'symphony.scv.profile-prepare-input.v1' for e in catalog['entries']))

    def portable_pack(self):
        desired = self.desired()
        desired['locators'][0]['format'] = 'json'
        source = self.owner('source_plan', {'operation_id': 'pack-source', 'current': None,
            'desired': desired, 'reason': 'Synthetic independently authored package fixture'})['source']
        capture = self.owner('capture_import', {'source': source, 'locator_id': 'docs',
            'resolved_uri': desired['locators'][0]['uri'], 'redirects': [], 'observed_at': '2026-09-10T12:00:00Z',
            'upstream_revision': None, 'media_type': 'application/json',
            'body': '{"plan":"fixture","default":3,"maximum":10}', 'completeness': 'complete', 'issues': []})
        value = {'type': 'integer', 'value': 10, 'unit': 'units'}
        expected = {'claim_id': 'maximum', 'subject': 'fixture-service', 'predicate': 'maximum',
            'scope': {'plan': 'fixture'}, 'statement_kind': 'documented_fact', 'dependencies': [], 'value': value}
        rule = {k: v for k, v in expected.items() if k != 'value'}
        rule.update(rule_id='max-rule', context=[{'pointer': '/plan', 'value': {'type': 'string', 'value': 'fixture', 'unit': None}}],
            extractor={'kind': 'json_pointer', 'pointer': '/maximum', 'type': 'integer', 'unit': 'units'})
        profile = {'protocol': 'symphony.scv.structured-profile.v1', 'profile_id': 'max-profile', 'profile_version': 'fixture-1',
            'provider_id': self.provider, 'source_id': 'fixture-docs', 'locator_id': 'docs', 'media_types': ['application/json'],
            'authored_by': 'Mapping fixture author', 'rationale': 'Map maximum separately from default', 'rules': [rule]}
        policy = {'policy_id': 'pack-fixture', 'max_age_seconds': 60, 'partial_capture': 'exclude', 'allowed_statement_kinds': ['documented_fact']}
        fixture = {'fixture_id': 'maximum-not-default', 'captures': [capture],
            'bindings': [{'profile_id': 'max-profile', 'capture_digest': capture['digest']}], 'selection_policy': policy}
        draft = {'protocol': 'symphony.scv.provider-pack.v1', 'pack_id': 'installed-fixture', 'pack_version': 'fixture-1',
            'authored_by': 'Independent package author', 'provenance': ['Synthetic conformance case, not provider capability'],
            'provider': {'provider_id': self.provider, 'family_id': self.family, 'display_name': 'Fixture', 'sources': [desired]},
            'profiles': [], 'structured_profiles': [profile], 'fixtures': [{'fixture_id': fixture['fixture_id'],
                'label': 'Maximum is 10, default is 3', 'authored_by': 'Independent expectation author',
                'rationale': 'Expected value authored before extraction; synthetic example', 'input_digest': None,
                'expected_claims': [expected], 'expected_extractions': [{'profile_id': 'max-profile', 'capture_digest': capture['digest'],
                    'rule_id': 'max-rule', 'claim_id': 'maximum', 'status': 'matched', 'reasons': []}]}]}
        pack = self.owner('provider_pack_prepare', {'pack': draft, 'fixtures': [fixture]})
        request = {'pack': pack, 'captures': [capture], 'bindings': fixture['bindings'], 'selection_policy': policy, 'fixtures': [fixture]}
        return request, self.owner('provider_pack_evaluate', request)

    def test_provider_pack_process_conformance_and_interface_receipt(self):
        if ARGS.version not in ('0.6.0-dev', '0.7.0-dev'):
            self.skipTest('portable provider package is an exact .6 addition')
        request, result = self.portable_pack()
        self.assertEqual(result['conformance'], {'passed': 1, 'failed': 0, 'not_run': 0})
        self.assertEqual(result['knowledge']['claims'][0]['value']['value'], 10)
        request['fixtures'] = []
        self.assertEqual(self.owner('provider_pack_evaluate', request)['conformance']['not_run'], 1)
        path = f'share/symphony/contracts/{self.module}/{ARGS.version}/OWNER-INTERFACE.json'
        owned = {item['path']: item for item in self.receipt['files']}
        self.assertIn(path, owned)
        manifest_bytes = (self.prefix / path).read_bytes()
        self.assertEqual(owned[path]['digest'], 'sha256:' + hashlib.sha256(manifest_bytes).hexdigest())
        self.assertEqual(len(json.loads(manifest_bytes)['operations']), 26)

    def test_composition_process_preserves_caller_choices_and_change_evidence(self):
        if ARGS.version not in ('0.6.0-dev', '0.7.0-dev'):
            self.skipTest('bounded composition is an exact .6 addition')
        _, pack = self.portable_pack()
        how = {'kind': 'adapter', 'reference': 'fixture.adapter.v1', 'description': 'Verify the caller-selected adapter'}
        recipe = {'recipe_id': 'first', 'provider_id': self.provider,
            'bindings': [{'requirement_id': 'capacity', 'claim': {'claim_id': 'maximum', 'subject': 'fixture-service', 'scope': {'plan': 'fixture'}}}],
            'prerequisites': [], 'requires_interfaces': [], 'supplies_interfaces': [], 'guarantee_changes': [],
            'implementation': {'status': 'unimplemented', 'reference': None}, 'resolution': how}
        requirement = {'requirement_id': 'capacity', 'importance': 'required', 'operator': 'gte',
            'right': {'kind': 'literal', 'value': {'type': 'integer', 'value': 8, 'unit': 'units'}}, 'resolution': how}
        request = {'interpretations': [], 'additional_knowledge': [], 'provider_packs': [pack],
            'query_time': '2026-09-10T12:00:01Z', 'requirements': [requirement],
            'slots': [{'slot_id': 'compute', 'allowed_provider_ids': [self.provider], 'recipes': [recipe]}],
            'allowed_guarantee_changes': [], 'counterfactuals': [], 'bounds': {'max_candidates': 2}}
        before = self.owner('composition_explore', request)
        candidate = before['scenarios'][0]['candidates'][0]
        self.assertEqual(candidate['status'], 'satisfied')
        self.assertEqual(candidate['implementation_status'], 'unimplemented')
        self.assertTrue(candidate['obligations'])
        request['query_time'] = '2026-09-10T12:02:00Z'
        after = self.owner('composition_explore', request)
        self.assertEqual(after['scenarios'][0]['candidates'][0]['status'], 'unresolved')
        changed = self.owner('composition_reassess', {'before': before, 'after': after})
        self.assertTrue(changed['change_axes']['query_time'])
        self.assertFalse(changed['change_axes']['evidence'])
        after['search']['eligible_combinations'] = 99
        unsigned = dict(after)
        unsigned.pop('digest')
        after['digest'] = digest(unsigned)
        proc, _ = self.invoke('composition_reassess', {'before': before, 'after': after})
        self.assertNotEqual(proc.returncode, 0)
        request['slots'][0]['allowed_provider_ids'] = ['unselected-provider']
        self.assertEqual(self.owner('composition_explore', request)['search']['stop_reason'], 'no_eligible_recipe')

    def test_source_transition_expected_state_and_digest(self):
        current = self.initial()
        desired = self.desired()
        desired['locators'][0]['uri'] = 'https://example.invalid/moved.md'
        _, proposal = self.invoke('source_plan', {'operation_id': 'move', 'current': current, 'desired': desired, 'reason': 'Fixture continuity'})
        plan = proposal['result']
        proc, applied = self.invoke('source_apply', {'plan': plan, 'current': current})
        self.assertEqual(proc.returncode, 0, applied)
        state = applied['result']['state']
        self.assertEqual(state['source_id'], current['source_id'])
        self.assertEqual(state['predecessor_digest'], current['digest'])
        proc, rejected = self.invoke('source_apply', {'plan': plan, 'current': state})
        self.assertNotEqual(proc.returncode, 0)
        self.assertEqual(rejected['error']['code'], 'scv.stale_state')

    def test_capture_provenance_and_graph_process(self):
        source = self.initial()
        body = '# Fixture\nSee [interface](https://example.invalid/interface).\n'
        proc, response = self.invoke('capture_import', {'source': source, 'locator_id': 'docs',
            'resolved_uri': source['locators'][0]['uri'], 'redirects': [], 'observed_at': '2026-09-10T12:00:00Z',
            'upstream_revision': None, 'media_type': 'text/markdown', 'body': body, 'completeness': 'complete', 'issues': []})
        self.assertEqual(proc.returncode, 0, response)
        capture = response['result']
        self.assertEqual(capture['body_digest'], 'sha256:' + hashlib.sha256(body.encode()).hexdigest())
        policy = {'policy_id': 'fixture-policy', 'max_age_seconds': None,
                  'allowed_statement_kinds': ['documented_fact', 'requirement', 'recommendation', 'observation', 'user_assertion', 'inference', 'hypothesis'],
                  'partial_capture': 'exclude'}
        proc, interpreted = self.invoke('knowledge_interpret', {'captures': [capture], 'claims': [], 'interpreter_version': 'fixture-v1', 'selection_policy': policy})
        self.assertEqual(proc.returncode, 0, interpreted)
        proc, graph = self.invoke('graph_build', {'knowledge': [interpreted['result']]})
        self.assertEqual(proc.returncode, 0, graph)
        proc, queried = self.invoke('graph_query', {'graph': graph['result'], 'query_time': '2026-09-10T12:00:01Z'})
        self.assertEqual(proc.returncode, 0, queried)
        self.assertEqual(queried['result']['graph_digest'], graph['result']['digest'])

    def test_protocol_rejection_is_machine_readable(self):
        for overrides in ({'target_engine': 'symphony-foreign'}, {'deadline_unix_ms': 0}, {'protocol': 'unsupported.v9'}):
            proc, response = self.invoke('inspect', {}, **overrides)
            self.assertNotEqual(proc.returncode, 0)
            self.assertEqual(response['outcome'], 'error')

    def capture(self, source, body, observed_at, completeness='complete'):
        proc, result = self.invoke('capture_import', {'source': source, 'locator_id': 'docs',
            'resolved_uri': source['locators'][0]['uri'], 'redirects': [], 'observed_at': observed_at,
            'upstream_revision': None, 'media_type': 'text/markdown', 'body': body,
            'completeness': completeness, 'issues': [] if completeness == 'complete' else ['fixture_fetch_failed']})
        self.assertEqual(proc.returncode, 0, result)
        return result['result']

    def owner(self, operation, payload):
        proc, response = self.invoke(operation, payload)
        self.assertEqual(proc.returncode, 0, response)
        return response['result']

    def test_provider_interpretation_connection_reassessment(self):
        source = self.initial()
        captured = self.capture(source, 'Fixture service. Port: 7844 UDP.', '2026-09-10T12:00:00Z')
        scope = {'service': 'fixture-v1'}
        profile = {'protocol': 'symphony.scv.interpretation-profile.v1', 'profile_id': 'fixture-profile',
            'profile_version': '1', 'provider_id': self.provider, 'source_id': source['source_id'],
            'locator_id': 'docs', 'media_types': ['text/markdown'], 'authored_by': 'synthetic test',
            'rationale': 'Local bounded extraction test', 'rules': [{'rule_id': 'port', 'claim_id': 'service-port',
            'subject': 'edge', 'predicate': 'requires-egress-port', 'scope': scope, 'statement_kind': 'requirement',
            'dependencies': [], 'context': ['Fixture service.'],
            'extractor': {'kind': 'delimited', 'prefix': 'Port: ', 'suffix': ' UDP.', 'type': 'integer', 'unit': 'port'}}]}
        profile['digest'] = digest(profile)
        policy = {'policy_id': 'fixture', 'max_age_seconds': 60,
            'allowed_statement_kinds': ['requirement'], 'partial_capture': 'exclude'}
        def interpret(cap):
            payload = {'captures': [cap], 'profiles': [profile], 'bindings': [{'profile_digest': profile['digest'],
                'capture_digest': cap['digest']}], 'selection_policy': policy}
            result = self.owner('provider_interpret', payload)
            self.assertEqual(result, self.owner('provider_interpret', payload))
            return result
        def evaluate(interpreted):
            return self.owner('connection_evaluate', {'interpretations': [interpreted], 'additional_knowledge': [],
                'query_time': '2026-09-10T12:00:01Z', 'connections': [{'connection_id': 'edge-node',
                'from_subject': 'edge', 'to_subject': 'node', 'checks': [{'check_id': 'port', 'importance': 'required',
                'left': {'claim_id': 'service-port', 'subject': 'edge', 'scope': scope}, 'operator': 'eq',
                'right': {'kind': 'literal', 'value': {'type': 'integer', 'value': 7844, 'unit': 'port'}}}]}]})
        interpreted = interpret(captured)
        before = evaluate(interpreted)
        self.assertEqual(before['connections'][0]['status'], 'satisfied')
        after = evaluate(interpret(self.capture(source, 'Fixture service. Port: 7845 UDP.', '2026-09-10T12:00:00Z')))
        self.assertEqual(after['connections'][0]['status'], 'contradicted')
        reassessed = self.owner('connection_reassess', {'before': before, 'after': after})
        self.assertTrue(reassessed['checks'][0]['changed'])
        self.assertTrue(reassessed['checks'][0]['affected'])
        self.assertTrue(reassessed['change_axes']['captures'])
        # Re-sealing a changed generated claim cannot bypass owner replay.
        forged = json.loads(json.dumps(before))
        forged['connections'][0]['status'] = 'contradicted'
        forged.pop('digest')
        forged['digest'] = digest(forged)
        proc, rejected = self.invoke('connection_reassess', {'before': forged, 'after': after})
        self.assertNotEqual(proc.returncode, 0)
        self.assertEqual(rejected['outcome'], 'error')
        missing = interpret(self.capture(source, 'Port: 7844 UDP.', '2026-09-10T12:00:00Z'))
        self.assertEqual(missing['extractions'][0]['status'], 'unresolved')
        self.assertEqual(evaluate(missing)['connections'][0]['status'], 'unresolved')

    def test_provider_coverage_process_binding_and_declared_gaps(self):
        if ARGS.version != '0.5.0-dev':
            self.skipTest('provider coverage is an exact .5 addition')
        source = self.initial()
        captured = self.capture(source, 'Fixture limit: 10 units.', '2026-09-10T12:00:00Z')
        profile = self.owner('profile_prepare', {'profile': {
            'protocol': 'symphony.scv.interpretation-profile.v1', 'profile_id': 'coverage-profile',
            'profile_version': 'fixture-1', 'provider_id': self.provider, 'source_id': source['source_id'],
            'locator_id': 'docs', 'media_types': ['text/markdown'], 'authored_by': 'Synthetic fixture',
            'rationale': 'A bounded literal extraction fixture, not live provider evidence',
            'rules': [{'rule_id': 'limit', 'claim_id': 'fixture-limit', 'subject': 'fixture-service',
                'predicate': 'maximum', 'scope': {}, 'statement_kind': 'documented_fact', 'dependencies': [],
                'context': [], 'extractor': {'kind': 'delimited', 'prefix': 'Fixture limit: ', 'suffix': ' units.',
                    'type': 'integer', 'unit': 'units'}}]}})
        interpreted = self.owner('provider_interpret', {'captures': [captured], 'profiles': [profile],
            'bindings': [{'profile_digest': profile['digest'], 'capture_digest': captured['digest']}],
            'selection_policy': {'policy_id': 'coverage-policy', 'max_age_seconds': None,
                'allowed_statement_kinds': ['documented_fact'], 'partial_capture': 'exclude'}})
        missing = self.desired()
        missing['source_id'] = 'not-acquired'
        declaration = self.owner('provider_onboard', {'provider_id': self.provider, 'family_id': self.family,
            'display_name': 'Fixture provider', 'sources': [self.desired(), missing]})
        capture_index = self.owner('capture_index', {'capture': captured})
        corpus = self.owner('corpus_build', {'corpus_id': 'coverage-corpus', 'previous': None,
            'snapshot_time': '2026-09-10T12:00:01Z', 'attempts': [{'member_id': 'docs', 'capture': capture_index}]})
        payload = {'provider': declaration, 'corpus_query': {'corpus': corpus, 'member_ids': [],
            'selection': 'latest_attempt', 'query_time': '2026-09-10T12:00:01Z', 'max_age_seconds': 60},
            'interpretations': [interpreted]}
        result = self.owner('provider_coverage', payload)
        self.assertEqual(result['protocol'], 'symphony.scv.provider-coverage.v1')
        self.assertEqual(result['domain'], ARGS.domain)
        self.assertEqual(result['input'], payload)
        self.assertEqual(result['summary']['declared_locators'], 2)
        self.assertEqual(result['summary']['unselected_declared_locators'], 1)
        self.assertEqual(result['summary']['matched_rule_attempts'], 1)
        self.assertEqual(result['summary']['replayed_selected_captures'], 1)
        selected_row = next(row for row in result['sources'] if row['source_id'] == source['source_id'])
        self.assertEqual(selected_row['declaration_match'], 'matches')
        metadata = self.owner('provider_coverage', {**payload, 'interpretations': []})
        self.assertEqual(metadata['summary']['replayed_selected_captures'], 0)
        forged = json.loads(json.dumps(interpreted))
        forged['extractions'][0]['status'] = 'unresolved'
        forged.pop('digest')
        forged['digest'] = digest(forged)
        proc, rejected = self.invoke('provider_coverage', {**payload, 'interpretations': [forged]})
        self.assertNotEqual(proc.returncode, 0, rejected)

    def test_corpus_refresh_retains_exact_historical_capture(self):
        source = self.initial()
        capture = self.capture(source, '# Retained evidence\n', '2026-09-10T12:00:00Z')
        first_index = self.owner('capture_index', {'capture': capture})
        first = self.owner('corpus_build', {'corpus_id': 'fixture', 'previous': None,
            'snapshot_time': '2026-09-10T12:00:01Z', 'attempts': [{'member_id': 'docs', 'capture': first_index}]})
        desired = self.desired()
        desired['locators'][0]['uri'] = 'https://example.invalid/moved.md'
        moved = self.owner('source_plan', {'operation_id': 'move', 'current': source, 'desired': desired, 'reason': 'fixture'})['source']
        failed = self.capture(moved, '', '2026-09-10T12:01:00Z', 'failed')
        failed_index = self.owner('capture_index', {'capture': failed})
        refreshed = self.owner('corpus_build', {'corpus_id': 'fixture', 'previous': first,
            'snapshot_time': '2026-09-10T12:01:01Z', 'attempts': [{'member_id': 'docs', 'capture': failed_index}]})
        member = refreshed['members'][0]
        self.assertEqual(member['latest_attempt'], failed_index)
        self.assertEqual(member['last_complete'], first_index)
        selected = self.owner('corpus_query', {'corpus': refreshed, 'member_ids': ['docs'],
            'selection': 'last_complete', 'max_age_seconds': 60, 'query_time': '2026-09-10T13:00:00Z'})
        result = selected['members'][0]
        self.assertEqual(result['selected']['capture_digest'], capture['digest'])
        self.assertEqual(result['freshness'], 'expired')
        self.assertFalse(result['source_revision_matches_latest'])
        self.assertEqual(refreshed['generation'], 2)
        self.assertEqual(refreshed['parent_digest'], first['digest'])

    def test_corpus_selection_removal_is_explicit(self):
        captured = self.capture(self.initial(), '# Fixture\n', '2026-09-10T12:00:00Z')
        index = self.owner('capture_index', {'capture': captured})
        before = self.owner('corpus_build', {'corpus_id': 'fixture', 'previous': None,
            'snapshot_time': '2026-09-10T12:00:01Z', 'attempts': [{'member_id': 'docs', 'capture': index}]})
        after = self.owner('corpus_build', {'corpus_id': 'fixture', 'previous': before,
            'snapshot_time': '2026-09-10T12:00:02Z', 'attempts': []})
        changed = self.owner('corpus_diff', {'before': before, 'after': after})
        self.assertEqual(changed['removed_member_ids'], ['docs'])
        self.assertEqual(changed['added_member_ids'], [])
        self.assertEqual(after['coverage']['requested_members'], 0)
        proc, response = self.invoke('corpus_query', {'corpus': after, 'member_ids': ['docs'],
            'selection': 'latest_attempt', 'max_age_seconds': None, 'query_time': '2026-09-10T13:00:00Z'})
        self.assertNotEqual(proc.returncode, 0, response)

    def test_duplicate_and_unknown_fields_rejected(self):
        value = json.dumps(self.request('inspect', {}))
        duplicate = value[:-1] + ',"payload":{}}'
        proc = subprocess.run([str(self.binary)], input=duplicate, text=True, capture_output=True, timeout=6, env={})
        self.assertNotEqual(proc.returncode, 0)
        self.assertEqual(json.loads(proc.stdout)['outcome'], 'error')
        proc, response = self.invoke('source_status', {'source': None, 'fabricated_permission': True})
        self.assertNotEqual(proc.returncode, 0)
        self.assertEqual(response['error']['code'], 'scv.fields')


if __name__ == '__main__':
    unittest.main(argv=['installed_integration.py'] + REST)
