"""Run only in an isolated runner: fixtures execute the workflow's real run blocks."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / '.github/workflows/linux-ci.yml'
DOCUMENT = yaml.load(WORKFLOW.read_text(), Loader=yaml.BaseLoader)
STEPS = {step['id']: step['run'] for step in DOCUMENT['jobs']['go']['steps'] if 'id' in step}

# Each mutation weakens one rejection while retaining its production command/token.
MUTANTS = {
    'invalid-runner-key': ('workflow', 'actionlint -shellcheck=',
                           "actionlint -ignore 'runs-onn|runs-on' -shellcheck=", 'test_workflow_invalid'),
    'nested-orphan': ('source', 'find . -name .git', 'find . -maxdepth 1 -name .git', 'test_nested_orphan'),
    'invalid-schema-type': ('schemas', 'Draft202012Validator.check_schema(schema)',
                            "if schema.get('type') == 'object': Draft202012Validator.check_schema(schema)",
                            'test_invalid_schema_type'),
    'verify-error': ('source', 'go mod verify', 'go mod verify || exit 0', 'test_invalid_module'),
    'build-error': ('source', 'go build ./...', 'go build ./... || exit 0', 'test_malformed_go'),
    'vet-error': ('source', 'go vet ./...', 'go vet ./... || exit 0', 'test_vet_failure'),
    'one-format-exception': ('source', 'test -z "$out"',
                             'test -z "$out" || test "$out" = unformatted.go', 'test_bad_formatting'),
    'skip-test-bodies': ('source', 'go test -v', "go test -run '^$' -v", 'test_failing_test'),
}


class LinuxCI(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='linux-ci-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = dict(os.environ, GOTOOLCHAIN='local', GOMAXPROCS='2', GOWORK='off', GOFLAGS='')

    def write(self, name, text):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)

    def run_step(self, name):
        script = STEPS[name]
        if mutant := os.environ.get('CI_MUTANT'):
            step, before, after, _ = MUTANTS[mutant]
            if step == name:
                self.assertEqual(script.count(before), 1, 'Mutation must apply exactly once')
                script = script.replace(before, after, 1)
        return subprocess.run(['bash', '--noprofile', '--norc', '-e', '-o', 'pipefail', '-c', script],
                              cwd=self.root, env=self.env, text=True, capture_output=True, timeout=120)

    def module(self):
        self.write('go.mod', 'module example.org/ci-fixture\n\ngo 1.26.0\n')
        self.write('valid.go', 'package fixture\n\nfunc Value() int { return 1 }\n')
        self.write('valid_test.go', 'package fixture\n\nimport "testing"\n\n'
                   'func TestValue(t *testing.T) {\n\tif Value() != 1 {\n\t\tt.Fatal("value")\n\t}\n}\n')

    # Proves the production entrypoint distinguishes source absence from nested orphan source.
    def test_spec_only(self):
        result = self.run_step('source')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('Spec-only bootstrap:', result.stdout)
        self.assertIn('tests did not run', result.stdout)

    def test_nested_orphan(self):
        self.write('nested/orphan.go', 'package orphan\n')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('Go source or module requires a root go.mod:', result.stderr)
        self.assertNotIn('Spec-only bootstrap:', result.stdout)

    def test_nested_module_without_root(self):
        self.write('nested/go.mod', 'module example.org/nested\n\ngo 1.26.0\n')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('Go source or module requires a root go.mod:', result.stderr)

    def test_symlink_orphan(self):
        (self.root / 'orphan.go').symlink_to('missing.go')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('Go source or module requires a root go.mod:', result.stderr)

    # Proves schema syntax, meta-schema constraints, and the missing-input refusal, not instance semantics.
    def test_existing_schema(self):
        self.write('schemas/workspace-result-1.schema.json',
                   (ROOT / 'schemas/workspace-result-1.schema.json').read_text())
        result = self.run_step('schemas')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_invalid_schema_type(self):
        self.write('schemas/invalid.schema.json', json.dumps({'type': 'not-a-type'}))
        result = self.run_step('schemas')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('SchemaError', result.stderr)

    def test_invalid_schema_json(self):
        self.write('schemas/invalid.schema.json', '{')
        result = self.run_step('schemas')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('JSONDecodeError', result.stderr)

    def test_missing_schema(self):
        result = self.run_step('schemas')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('No draft schemas found', result.stderr)

    # Real, dependency-free Go fixtures prove each failure propagates through the production block.
    # Module verification covers malformed go.mod; downloaded-module tampering is outside this fixture.
    def test_valid_module(self):
        self.module()
        result = self.run_step('source')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('PASS: TestValue', result.stdout)
        self.assertNotIn('Spec-only bootstrap:', result.stdout)

    def test_invalid_module(self):
        self.module()
        self.write('go.mod', 'not a module file\n')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('go.mod:', result.stderr)

    # A broken module link must fail as a module read, never become source absence.
    def test_broken_module_link(self):
        (self.root / 'go.mod').symlink_to('missing.mod')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertNotIn('Spec-only bootstrap:', result.stdout)

    def test_malformed_go(self):
        self.module()
        self.write('valid.go', 'package fixture\n\nfunc Value(\n')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('syntax error', result.stderr)

    def test_vet_failure(self):
        self.module()
        self.write('vet.go', 'package fixture\n\nimport "fmt"\n\n'
                   'func BadPrint() { fmt.Printf("%d", "wrong") }\n')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('fmt.Printf format %d', result.stderr)

    def test_bad_formatting(self):
        self.module()
        self.write('unformatted.go', 'package fixture\nfunc Other()int{return 2}\n')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('unformatted.go', result.stdout)

    def test_failing_test(self):
        self.module()
        self.write('valid_test.go', 'package fixture\n\nimport "testing"\n\n'
                   'func TestValue(t *testing.T) { t.Fatal("fixture failure") }\n')
        result = self.run_step('source')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('fixture failure', result.stdout)

    # The same actionlint entrypoint accepts the candidate and rejects an invalid workflow key.
    def test_workflow_valid(self):
        self.write('.github/workflows/linux-ci.yml', WORKFLOW.read_text())
        result = self.run_step('workflow')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_workflow_invalid(self):
        self.write('.github/workflows/linux-ci.yml', WORKFLOW.read_text().replace('runs-on:', 'runs-onn:', 1))
        result = self.run_step('workflow')
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('runs-onn', result.stdout + result.stderr)

    # Checks trigger/action/permission data, not hosted scheduling, token authority, or runner execution.
    def test_workflow_contract(self):
        self.assertEqual(set(DOCUMENT['on']), {'pull_request', 'push'})
        self.assertEqual(DOCUMENT['on']['push']['branches'], ['main'])
        self.assertEqual(DOCUMENT['permissions'], {'contents': 'read'})
        job = DOCUMENT['jobs']['go']
        self.assertEqual(job['runs-on'], 'ubuntu-24.04')
        self.assertNotIn('continue-on-error', job)
        for step in job['steps']:
            self.assertNotIn('continue-on-error', step)
            self.assertNotIn('if', step)
            if 'uses' in step:
                self.assertRegex(step['uses'], r'^[^@]+@[0-9a-f]{40}$')
        self.assertEqual(job['steps'][0]['with']['persist-credentials'], 'false')

    # Every named control must apply and produce an assertion failure in its named behavioral test.
    @unittest.skipIf('CI_MUTANT' in os.environ, 'Do not recursively launch controls')
    def test_narrowing_mutants(self):
        for mutant, (_, _, _, test) in MUTANTS.items():
            with self.subTest(mutant=mutant):
                result = subprocess.run([sys.executable, str(Path(__file__).resolve()),
                                         f'LinuxCI.{test}', '-v'], cwd=ROOT,
                                        env=dict(os.environ, CI_MUTANT=mutant), text=True,
                                        capture_output=True, timeout=180)
                self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
                self.assertIn(f'FAIL: {test} ', result.stderr, result.stdout + result.stderr)
                print(f'Killed {mutant}: {test}', flush=True)


if __name__ == '__main__':
    unittest.main()
