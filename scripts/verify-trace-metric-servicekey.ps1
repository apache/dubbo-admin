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

# Keep this entry point for PowerShell users; Python owns all contract checks
# and the single fixture matrix executed by the real Collector.
param(
    [string]$Python = 'python',
    [string]$Collector
)

$ErrorActionPreference = 'Stop'
$arguments = @((Join-Path $PSScriptRoot 'verify-trace-metric-collector.py'))
if ($Collector) {
    $arguments += @('--collector', $Collector)
} else {
    $arguments += '--static-only'
}
& $Python @arguments
if ($LASTEXITCODE -ne 0) {
    throw "Trace-to-metric verification failed (exit $LASTEXITCODE)"
}
