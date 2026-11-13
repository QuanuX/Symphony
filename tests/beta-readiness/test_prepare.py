#!/usr/bin/env python3
"""Focused scope/recipe drift regressions; no final acceptance fixtures."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('beta_prepare', Path(__file__).with_name('prepare.py'))
beta = importlib.util.module_from_spec(spec)
spec.loader.exec_module(beta)

class ScopeTests(unittest.TestCase):
    def setUp(self):
        def read(path): return json.loads((ROOT / path).read_text())
        self.packages = read('docs/public-beta/package-plan.json')
        self.deps = read('cmake/mvp-dependencies.json')
        self.identities = read('cmake/mvp-release-identities.json')
        self.plan = read('tests/launch-acceptance/plan.json')
        self.mapping = read('docs/public-beta/acceptance-mapping.json')

    def builds(self):
        return beta.construction(self.packages, self.deps, self.identities, self.plan)

    def first_core(self):
        return next(r for r in self.packages['package_recipes'] if r['id'] == 'PKG-CORE-' + self.plan['required_profiles'][0])

    def first_native(self):
        return next(r for r in self.packages['package_recipes'] if r['id'] == 'PKG-NATIVE-knowledge-vector-engine-cpp')

    def test_complete_selection_remains_plan_only(self):
        result = self.builds()
        self.assertEqual(4, len(result['profiles']))
        for row in result['profiles'].values():
            self.assertEqual(51, len(row['native_build_order']))
            self.assertEqual(6, len(row['go_commands']))
            self.assertIsNone(row['candidate_identity'])
            self.assertIsNone(row['retrieved_anaconda_identity'])
        self.assertFalse(result['release_acceptance_complete'])
        self.assertFalse(result['publication_authorized'])

    def test_missing_core_component(self):
        self.first_core()['content_recipe_ids'].pop()
        with self.assertRaises(ValueError): self.builds()

    def test_duplicate_core_component(self):
        selected = self.first_core()['content_recipe_ids']; selected.append(selected[0])
        with self.assertRaises(ValueError): self.builds()

    def test_missing_dependency(self):
        self.first_core()['content_recipe_ids'].remove('PKG-NATIVE-knowledge-vector-engine-cpp')
        with self.assertRaisesRegex(ValueError, 'missing native dependency'): self.builds()

    def test_version_drift(self):
        self.first_native()['proposed_package_version'] = '9.0.0'
        with self.assertRaises(ValueError): self.builds()

    def test_installed_option_drift(self):
        self.first_native()['installed_options'] = {'SOURCE_FALLBACK': 'ON'}
        with self.assertRaises(ValueError): self.builds()

    def test_private_target_substitution(self):
        self.first_native()['build_targets'] = ['replacement']
        with self.assertRaises(ValueError): self.builds()

    def test_go_version_drift(self):
        row = next(r for r in self.packages['package_recipes'] if r['id'] == 'PKG-GO-qxctl')
        row['proposed_package_version'] = '9.0.0'
        with self.assertRaisesRegex(ValueError, 'current Go owner'): self.builds()

    def test_go_entrypoint_substitution(self):
        row = next(r for r in self.packages['package_recipes'] if r['id'] == 'PKG-GO-qxctl')
        row['main_package'] = './replacement'
        with self.assertRaisesRegex(ValueError, 'current Go owner'): self.builds()

    def test_profile_omission(self):
        self.first_native()['candidate_profiles'] = []
        with self.assertRaises(ValueError): self.builds()

    def test_order_inversion(self):
        self.deps['native_build_order'].reverse()
        with self.assertRaisesRegex(ValueError, 'non-topological'): self.builds()

    def test_order_omission(self):
        self.deps['native_build_order'].pop(0)
        with self.assertRaises(ValueError): self.builds()

    def test_duplicate_recipe(self):
        self.packages['package_recipes'].append(copy.deepcopy(self.first_native()))
        with self.assertRaises(ValueError): self.builds()

    def workorders(self):
        return beta.workorders(self.mapping, self.plan)

    def test_all_requirements_stay_pending(self):
        out = self.workorders()
        self.assertEqual([36, 58, 31], [len(out[k]) for k in ['core', 'paper', 'supplemental']])
        for kind in ['core', 'paper', 'supplemental']:
            for row in out[kind]:
                self.assertEqual('pending', row['acceptance_status'])
                self.assertIsNone(row['final_artifact_bindings'])
                self.assertIsNone(row['observed_result'])

    def test_criterion_deletion(self):
        self.mapping['paper_criteria'].pop()
        with self.assertRaises(ValueError): self.workorders()

    def test_criterion_duplication(self):
        self.mapping['paper_criteria'][1] = copy.deepcopy(self.mapping['paper_criteria'][0])
        with self.assertRaises(ValueError): self.workorders()

    def test_cloudflare_operation_deletion(self):
        self.mapping['cloudflare_operation_cells'].pop()
        with self.assertRaises(ValueError): self.workorders()

    def test_coordinated_profile_scope_reduction(self):
        self.plan['required_profiles'].pop()
        self.mapping['required_profiles'] = self.plan['required_profiles'][:]
        with self.assertRaises(ValueError): self.workorders()

    def test_coordinated_scenario_scope_reduction(self):
        removed = self.plan['required_scenarios'].pop()
        self.mapping['required_scenarios'] = self.plan['required_scenarios'][:]
        self.mapping['campaigns'] = [c for c in self.mapping['campaigns'] if c['scenario'] != removed]
        with self.assertRaises(ValueError): self.workorders()

    def test_scenario_deletion(self):
        self.mapping['campaigns'].pop()
        with self.assertRaises(ValueError): self.workorders()

if __name__ == '__main__': unittest.main(verbosity=2)
