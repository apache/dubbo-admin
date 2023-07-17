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
WORKDIR=$(cd "$WORKDIR"; pwd)

set -eux

# script set up the plain text rendered

KUBERNETES=${WORKDIR}/../kubernetes
DASHBOARDS=${WORKDIR}
mkdir -p "${KUBERNETES}"
kubectl delete ns dubbo-system && kubectl create ns dubbo-system


# Set up zookeeper
helm template zookeeper zookeeper \
  --namespace dubbo-system \
  --version 11.1.6 \
  --repo https://charts.bitnami.com/bitnami  \
  -f "${WORKDIR}/values-zookeeper.yaml" \
  > "${KUBERNETES}/zookeeper.yaml"


# Set up prometheus
helm template prometheus prometheus \
  --namespace dubbo-system \
  --version 20.0.2 \
  --repo https://prometheus-community.github.io/helm-charts \
  -f "${WORKDIR}/values-prometheus.yaml" \
  > "${KUBERNETES}/prometheus.yaml"


# Set up grafana
{
  helm template grafana grafana \
    --namespace dubbo-system \
    --version 6.52.4 \
    --repo https://grafana.github.io/helm-charts \
    -f "${WORKDIR}/values-grafana.yaml" \

  echo -e "\n---\n"

  kubectl create configmap -n dubbo-system admin-extra-dashboards \
    --dry-run=client -oyaml \
    --from-file=extra-dashboard.json="${DASHBOARDS}/dashboards/external-dashboard.json"
} > "${KUBERNETES}/grafana.yaml"


# Set up skywalking
helm template skywalking skywalking \
  --namespace dubbo-system \
  --version 4.3.0 \
  --repo https://apache.jfrog.io/artifactory/skywalking-helm \
  -f "${WORKDIR}/values-skywalking.yaml" \
  > "${KUBERNETES}/skywalking.yaml"


# Set up zipkin
helm template zipkin zipkin \
  --namespace dubbo-system \
  --version 0.3.0 \
  --repo https://openzipkin.github.io/zipkin \
  -f "${WORKDIR}/values-zipkin.yaml" \
  > "${KUBERNETES}/zipkin.yaml"