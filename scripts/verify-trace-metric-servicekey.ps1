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

$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$collector = Get-Content -LiteralPath (Join-Path $root 'release/kubernetes/monitoring/otelCollector-configuration.yaml') -Raw
$datasource = Get-Content -LiteralPath (Join-Path $root 'release/kubernetes/monitoring/grafana-dashboardDatasources.yaml') -Raw

# Guard the exact patterns used by the Collector. This is a static contract test;
# the Collector config is validated separately with otelcol-contrib.
$pipeline = [regex]::Match($collector, '(?ms)^    service:\s*\r?\n      pipelines:\s*\r?\n        traces:\s*\r?\n(?<body>.*?)(?=^        \S|^    \S|\z)')
if (-not $pipeline.Success -or
    -not [regex]::IsMatch($pipeline.Groups['body'].Value, '(?m)^          processors:\s*\[\s*k8sattributes\s*,\s*transform/dubbo_application\s*\]\s*$')) {
    throw 'Trace pipeline must execute k8sattributes followed by transform/dubbo_application'
}
'PASS TRACE_PIPELINE_WIRING'
foreach ($statement in @(
    'set(attributes["dubbo.metric.interface"], attributes["rpc.service"])',
    'replace_pattern(attributes["dubbo.metric.interface"], "^[^/]+/", "")',
    'set(attributes["dubbo.metric.version"], attributes["dubbo.metric.interface"])',
    'replace_pattern(attributes["dubbo.metric.interface"], ":[^:]+$", "")',
    'replace_pattern(attributes["dubbo.metric.group"], "/.*$", "")',
    'replace_pattern(attributes["dubbo.metric.version"], "^.*:", "")',
    'set(attributes["dubbo.metric.group"], attributes["dubbo.group"])',
    'set(attributes["dubbo.metric.version"], attributes["dubbo.version"])'
)) {
    if (-not $collector.Contains($statement)) { throw "Missing Collector statement: $statement" }
}
$stripGroup = $collector.IndexOf('replace_pattern(attributes["dubbo.metric.interface"], "^[^/]+/", "")')
$deriveVersion = $collector.IndexOf('set(attributes["dubbo.metric.version"], attributes["dubbo.metric.interface"])')
$stripVersion = $collector.IndexOf('replace_pattern(attributes["dubbo.metric.interface"], ":[^:]+$", "")')
if (-not ($stripGroup -lt $deriveVersion -and $deriveVersion -lt $stripVersion)) {
    throw 'Collector must remove group before deriving version, then remove version from interface'
}
if (-not $datasource.Contains('{"key": "dubbo.metric.interface", "value": "interface"}')) {
    throw 'Grafana interface mapping is not normalized'
}
if ($datasource.Contains('{"key": "rpc.service", "value": "interface"}')) {
    throw 'Grafana still maps raw rpc.service to interface'
}
if ($datasource.Contains('{"key": "dubbo.metric.group", "value": "group"}') -or
    $datasource.Contains('{"key": "dubbo.metric.version", "value": "version"}')) {
    throw 'Optional group/version matcher mapping requires separate verification'
}
foreach ($line in ($collector -split "`r?`n" | Where-Object { $_ -match '^\s+- (set|replace_pattern)\(attributes\["dubbo\.metric\.' })) {
    if (-not $line.Contains('attributes["rpc.system"] == "apache_dubbo"')) {
        throw "Normalization statement lacks the Dubbo guard: $line"
    }
}

function Normalize-ForStaticTest([hashtable]$span) {
    $result = $span.Clone()
    $service = [string]$span['rpc.service']
    if ($span['rpc.system'] -cne 'apache_dubbo' -or [string]::IsNullOrEmpty($service)) { return $result }
    $remainder = $service -creplace '^[^/]+/', ''
    $result['dubbo.metric.interface'] = $remainder -creplace ':[^:]+$', ''
    if ($span.ContainsKey('dubbo.group')) {
        if (-not [string]::IsNullOrEmpty([string]$span['dubbo.group'])) { $result['dubbo.metric.group'] = $span['dubbo.group'] }
    } elseif ($service -cmatch '^[^/]+/.+') {
        $result['dubbo.metric.group'] = $service -creplace '/.*$', ''
    }
    if ($span.ContainsKey('dubbo.version')) {
        if (-not [string]::IsNullOrEmpty([string]$span['dubbo.version'])) { $result['dubbo.metric.version'] = $span['dubbo.version'] }
    } elseif ($remainder -cmatch ':[^:]+$') {
        $result['dubbo.metric.version'] = $remainder -creplace '^.*:', ''
    }
    return $result
}

$cases = @(
    @{ Name='PURE_INTERFACE'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='org.example.EchoService' }; Interface='org.example.EchoService'; Group=$null; Version=$null },
    @{ Name='GROUP_ONLY'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='gray/org.example.EchoService' }; Interface='org.example.EchoService'; Group='gray'; Version=$null },
    @{ Name='VERSION_ONLY'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='org.example.EchoService:1.0.0' }; Interface='org.example.EchoService'; Group=$null; Version='1.0.0' },
    @{ Name='GROUP_AND_VERSION'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='gray/org.example.EchoService:1.0.0' }; Interface='org.example.EchoService'; Group='gray'; Version='1.0.0' },
    @{ Name='COLON_IN_GROUP_NO_VERSION'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='blue:canary/org.example.EchoService' }; Interface='org.example.EchoService'; Group='blue:canary'; Version=$null },
    @{ Name='COLON_IN_GROUP_WITH_VERSION'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='blue:canary/org.example.EchoService:1.0.0' }; Interface='org.example.EchoService'; Group='blue:canary'; Version='1.0.0' },
    @{ Name='POST_3724_EXPLICIT_ATTRIBUTES'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='org.example.EchoService'; 'dubbo.group'='gray'; 'dubbo.version'='1.0.0' }; Interface='org.example.EchoService'; Group='gray'; Version='1.0.0' },
    @{ Name='EXPLICIT_ATTRIBUTE_PRECEDENCE'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='gray/org.example.EchoService:1.0.0'; 'dubbo.group'='blue'; 'dubbo.version'='2.0.0' }; Interface='org.example.EchoService'; Group='blue'; Version='2.0.0' },
    @{ Name='JAVA_PURE_INTERFACE'; Input=@{ 'rpc.system'='apache_dubbo'; 'rpc.service'='org.apache.dubbo.samples.OrderService' }; Interface='org.apache.dubbo.samples.OrderService'; Group=$null; Version=$null }
)
foreach ($case in $cases) {
    $actual = Normalize-ForStaticTest $case.Input
    if ($actual['dubbo.metric.interface'] -cne $case.Interface -or
        ($null -eq $case.Group -and $actual.ContainsKey('dubbo.metric.group')) -or
        ($null -ne $case.Group -and $actual['dubbo.metric.group'] -cne $case.Group) -or
        ($null -eq $case.Version -and $actual.ContainsKey('dubbo.metric.version')) -or
        ($null -ne $case.Version -and $actual['dubbo.metric.version'] -cne $case.Version) -or
        $actual['rpc.service'] -cne $case.Input['rpc.service']) {
        throw "$($case.Name) failed: $($actual | ConvertTo-Json -Compress)"
    }
    "PASS $($case.Name)"
}
$nonDubbo = Normalize-ForStaticTest @{ 'rpc.system'='http'; 'rpc.service'='org.example.EchoService' }
if ($nonDubbo.ContainsKey('dubbo.metric.interface') -or $nonDubbo.ContainsKey('dubbo.metric.group') -or $nonDubbo.ContainsKey('dubbo.metric.version')) {
    throw 'NON_DUBBO_UNCHANGED failed'
}
'PASS NON_DUBBO_UNCHANGED'
