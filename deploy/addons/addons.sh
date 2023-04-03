#!/usr/bin/env bash

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

WORKDIR=$(dirname "$0")
DASHBOARDS="${WORKDIR}/dashboards"
TMP=$(mktemp -d)

set -eux

# Set up prometheus
helm template prometheus prometheus \
  --namespace default \
  --version 20.0.2 \
  --repo https://prometheus-community.github.io/helm-charts \
  -f "${WORKDIR}/values-prometheus.yaml"

function extraDashboard() {
  < "${DASHBOARDS}/$1" jq -c  > "${TMP}/$1"
}

# Set up grafana
{
  helm template grafana grafana \
    --namespace default \
    --version 6.52.4 \
    --repo https://grafana.github.io/helm-charts \
    -f "${WORKDIR}/values-grafana.yaml"

  extraDashboard "external-dashboard.json"

  kubectl create configmap -n default external-dashboard \
    --dry-run=client -oyaml \
    --from-file=external-dashboard.json="${TMP}/external-dashboard.json"
}

# Set up sw
{
  helm template skywalking skywalking \
  --namespace default \
  --version 4.3.0 \
  --repo https://apache.jfrog.io/artifactory/skywalking-helm \
  -f "${WORKDIR}/values-sw.yaml"
}

# Set up zipkin
{
  helm template zipkin zipkin \
  --namespace default \
  --version 0.3.0 \
  --repo https://openzipkin.github.io/zipkin \
  -f "${WORKDIR}/values-zipkin.yaml"
}
