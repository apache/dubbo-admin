# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Execute the release transform in Collector 0.112.0, without cluster access.

Requires Python 3, PyYAML and a native otelcol-contrib 0.112.0 executable.
Only the isolated HTTP receiver on loopback and a temporary file exporter run.
"""

import argparse
import json
from pathlib import Path
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

import yaml


def attributes(values):
    return [{"key": key, "value": {"stringValue": value}} for key, value in values.items()]


def read_spans(path):
    spans = {}
    if not path.exists():
        return spans
    for line in path.read_text(encoding="utf-8").splitlines():
        try:
            batch = json.loads(line)
        except json.JSONDecodeError:
            continue  # The file exporter may still be flushing its last line.
        for resource in batch.get("resourceSpans", []):
            for scope in resource.get("scopeSpans", []):
                for span in scope.get("spans", []):
                    spans[span["name"]] = {
                        item["key"]: item["value"].get("stringValue")
                        for item in span.get("attributes", [])
                    }
    return spans


def verify(collector, manifest):
    version = subprocess.check_output([collector, "--version"], text=True).strip()
    if not version.endswith(" 0.112.0"):
        raise RuntimeError(f"Expected Collector Contrib 0.112.0, got {version}")
    config = yaml.safe_load(yaml.safe_load(manifest.read_text(encoding="utf-8"))["data"]["config.yaml"])
    pipeline = config["service"]["pipelines"]["traces"]["processors"]
    if pipeline != ["k8sattributes", "transform/dubbo_application"]:
        raise RuntimeError("Trace pipeline must execute k8sattributes then transform/dubbo_application")
    transform = config["processors"]["transform/dubbo_application"]

    # Expected outputs are fixtures, not a second implementation of the transform.
    cases = [
        ("PURE_INTERFACE", "org.example.EchoService", {}, "org.example.EchoService", None, None),
        ("GROUP_ONLY", "gray/org.example.EchoService", {}, "org.example.EchoService", "gray", None),
        ("VERSION_ONLY", "org.example.EchoService:1.0.0", {}, "org.example.EchoService", None, "1.0.0"),
        ("GROUP_AND_VERSION", "gray/org.example.EchoService:1.0.0", {}, "org.example.EchoService", "gray", "1.0.0"),
        ("COLON_IN_GROUP", "blue:canary/org.example.EchoService:1.0.0", {}, "org.example.EchoService", "blue:canary", "1.0.0"),
        ("EXPLICIT_ATTRIBUTES", "org.example.EchoService", {"dubbo.group": "gray", "dubbo.version": "1.0.0"}, "org.example.EchoService", "gray", "1.0.0"),
        ("EXPLICIT_PRECEDENCE", "gray/org.example.EchoService:1.0.0", {"dubbo.group": "blue", "dubbo.version": "2.0.0"}, "org.example.EchoService", "blue", "2.0.0"),
        ("JAVA_INTERFACE", "org.apache.dubbo.samples.OrderService", {}, "org.apache.dubbo.samples.OrderService", None, None),
    ]
    fixtures = []
    for name, service, extra, interface, group, version in cases:
        source = {"rpc.system": "apache_dubbo", "rpc.service": service, "rpc.method": "echo", **extra}
        expected = {"dubbo.application": "fixture-app", "dubbo.metric.interface": interface,
                    "dubbo.metric.group": group, "dubbo.metric.version": version}
        fixtures.append((name, source, {"service.name": "fixture-app"}, expected))
    for name, source in [
        ("NON_DUBBO", {"rpc.system": "http", "rpc.service": "example", "rpc.method": "GET"}),
        ("MISSING_SERVICE", {"rpc.system": "apache_dubbo", "rpc.method": "echo"}),
    ]:
        fixtures.append((name, source, {"service.name": "fixture-app"}, {}))
    fixtures.append(("NO_RESOURCE_NAME", {"rpc.system": "apache_dubbo", "rpc.service": "example", "rpc.method": "echo"}, {},
                     {"dubbo.metric.interface": "example"}))

    payload = {"resourceSpans": []}
    for index, (name, source, resource, _) in enumerate(fixtures, 1):
        payload["resourceSpans"].append({
            "resource": {"attributes": attributes(resource)},
            "scopeSpans": [{"scope": {"name": "trace-metric-contract"}, "spans": [{
                "traceId": f"{index:032x}", "spanId": f"{index:016x}", "name": name, "kind": 2,
                "startTimeUnixNano": "1700000000000000000", "endTimeUnixNano": "1700000000001000000",
                "attributes": attributes(source),
            }]}],
        })

    with tempfile.TemporaryDirectory(prefix="dubbo-trace-metric-") as directory:
        root = Path(directory)
        release_config = root / "release.yaml"
        release_config.write_text(yaml.safe_dump(config), encoding="utf-8")
        subprocess.run([collector, "validate", f"--config={release_config}"], check=True)
        print("PASS RELEASE_COLLECTOR_CONFIG")
        with socket.socket() as reservation:
            reservation.bind(("127.0.0.1", 0))
            port = reservation.getsockname()[1]
        output = root / "traces.json"
        isolated = {
            "receivers": {"otlp": {"protocols": {"http": {"endpoint": f"127.0.0.1:{port}"}}}},
            "processors": {"transform/dubbo_application": transform},
            "exporters": {"file": {"path": str(output)}},
            "service": {"pipelines": {"traces": {"receivers": ["otlp"],
                "processors": ["transform/dubbo_application"], "exporters": ["file"]}}},
        }
        isolated_config = root / "isolated.yaml"
        isolated_config.write_text(yaml.safe_dump(isolated), encoding="utf-8")
        subprocess.run([collector, "validate", f"--config={isolated_config}"], check=True)
        with (root / "collector.log").open("w", encoding="utf-8") as log:
            process = subprocess.Popen([collector, f"--config={isolated_config}"], stdout=log, stderr=log)
            try:
                deadline = time.monotonic() + 20
                while True:
                    if process.poll() is not None:
                        raise RuntimeError("Isolated Collector exited before accepting traces")
                    request = urllib.request.Request(f"http://127.0.0.1:{port}/v1/traces",
                        json.dumps(payload).encode(), {"Content-Type": "application/json"})
                    try:
                        # Bypass host proxies: the receiver is strictly local.
                        with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(request, timeout=2) as response:
                            result = json.load(response)
                        if result.get("partialSuccess"):
                            raise RuntimeError(f"Collector rejected fixture spans: {result}")
                        break
                    except urllib.error.HTTPError:
                        raise
                    except urllib.error.URLError:
                        if time.monotonic() >= deadline:
                            raise RuntimeError("Timed out waiting for isolated Collector")
                        time.sleep(0.1)
                while time.monotonic() < deadline:
                    if len(read_spans(output)) == len(fixtures):
                        break
                    time.sleep(0.1)
                else:
                    raise RuntimeError("Timed out waiting for file-exported spans")
            finally:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
        actual = read_spans(output)
        for name, source, _, expected in fixtures:
            values = actual[name]
            for key, value in source.items():
                if values.get(key) != value:
                    raise AssertionError(f"{name}: original {key} changed")
            for key in ("dubbo.application", "dubbo.metric.interface", "dubbo.metric.group", "dubbo.metric.version"):
                value = expected.get(key)
                if (value is None and key in values) or (value is not None and values.get(key) != value):
                    raise AssertionError(f"{name}: {key}: expected {value!r}, got {values.get(key)!r}")
            print(f"PASS COLLECTOR_TRANSFORM_{name}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--collector", required=True, help="Native otelcol-contrib 0.112.0 executable")
    parser.add_argument("--manifest", type=Path, default=Path(__file__).resolve().parents[1] /
                        "release/kubernetes/monitoring/otelCollector-configuration.yaml")
    options = parser.parse_args()
    verify(options.collector, options.manifest)
