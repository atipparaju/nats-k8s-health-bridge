#!/usr/bin/env python3
"""Offline Helm template integration checks; no cluster, registry or secrets required."""
import subprocess
import unittest
from pathlib import Path

CHART = Path(__file__).resolve().parents[1] / "helm/nats-k8s-health-bridge"


def render(*settings):
    command = ["helm", "template", "issue43", str(CHART)]
    for key, value in settings:
        command += ["--set-string", f"{key}={value}"]
    return subprocess.run(command, text=True, capture_output=True, check=False)


def example(output):
    # This YAML is stored in a ConfigMap as documentation, not a deployed workload.
    assert "  sidecar-example.yaml: |" in output
    return output.split("  sidecar-example.yaml: |", 1)[1]


class HelmSidecarExampleTests(unittest.TestCase):
    def test_default_is_safe_and_not_a_workload(self):
        result = render()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.count("kind: ConfigMap"), 1)
        self.assertNotIn("\nkind: Deployment\n", result.stdout)
        sample = example(result.stdout)
        for expected in ('nats-k8s-health-bridge:local', "imagePullPolicy: Never",
                         "readOnlyRootFilesystem: true", "allowPrivilegeEscalation: false",
                         'drop: ["ALL"]', "type: RuntimeDefault", "runAsNonRoot: true",
                         "path: /healthz", "path: /readyz", "readOnly: true"):
            self.assertIn(expected, sample)
        self.assertNotIn("ghcr.io/", result.stdout)
        self.assertNotIn(":latest", result.stdout)
        self.assertNotIn("secretKeyRef:", sample)
        self.assertNotIn("secretName:", sample)
        self.assertNotIn("NKHB_NATS_CREDENTIALS_FILE", sample)

    def test_existing_credentials_and_tokens_are_wired(self):
        result = render(("nats.credentialsSecret", "nats-creds"),
                        ("server.metricsTokenSecret", "metrics-auth"),
                        ("server.statusTokenSecret", "status-auth"))
        self.assertEqual(result.returncode, 0, result.stderr)
        sample = example(result.stdout)
        for expected in ("NKHB_NATS_CREDENTIALS_FILE", "/etc/nats-health-bridge-creds/user.creds",
                         'secretName: "nats-creds"', "key: user.creds",
                         "NKHB_SERVER_METRICS_TOKEN", 'name: "metrics-auth"',
                         "NKHB_SERVER_STATUS_TOKEN", 'name: "status-auth"'):
            self.assertIn(expected, sample)
        self.assertNotIn("nats-health-bridge-tls", sample)

    def test_ca_only_tls_mounts_only_ca(self):
        result = render(("nats.tlsSecret", "nats-tls"))
        self.assertEqual(result.returncode, 0, result.stderr)
        sample = example(result.stdout)
        self.assertIn("NKHB_NATS_TLS_CA", sample)
        self.assertIn("key: ca.crt", sample)
        self.assertNotIn("NKHB_NATS_TLS_CERT", sample)
        self.assertNotIn("key: tls.crt", sample)

    def test_client_tls_mounts_pair_and_ca(self):
        result = render(("nats.tlsSecret", "nats-tls"),
                        ("nats.tlsClientAuth", "true"))
        self.assertEqual(result.returncode, 0, result.stderr)
        sample = example(result.stdout)
        for expected in ("NKHB_NATS_TLS_CA", "NKHB_NATS_TLS_CERT", "NKHB_NATS_TLS_KEY",
                         "key: ca.crt", "key: tls.crt", "key: tls.key"):
            self.assertIn(expected, sample)

    def test_nkey_alternative(self):
        result = render(("nats.nkeySecret", "nats-nkey"))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("/etc/nats-health-bridge-nkey/user.nkey", result.stdout)
        self.assertIn("key: user.nkey", result.stdout)
        self.assertNotIn("NKHB_NATS_CREDENTIALS_FILE", result.stdout)

    def test_invalid_or_conflicting_values_fail_render(self):
        failures = [
            (("nats.tlsClientAuth", "true"),),
            (("nats.credentialsSecret", "creds"), ("nats.nkeySecret", "nkey")),
            (("nats.credentialsSecret", "invalid@name"),),
            (("server.statusTokenSecret", "BadName"),),
        ]
        for values in failures:
            with self.subTest(values=values):
                result = render(*values)
                self.assertNotEqual(result.returncode, 0, result.stdout)


if __name__ == "__main__":
    unittest.main()
