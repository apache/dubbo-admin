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

set -euo pipefail

namespace=dubbo-system
secret_name=dubbo-admin-auth
manifest_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for command in kubectl openssl; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'Required command not found: %s\n' "$command" >&2
    exit 1
  fi
done

if ! kubectl get namespace "$namespace" >/dev/null 2>&1; then
  kubectl create namespace "$namespace"
fi

if kubectl -n "$namespace" get secret "$secret_name" >/dev/null 2>&1; then
  secret_value="$(kubectl -n "$namespace" get secret "$secret_name" -o jsonpath='{.data.session-secret}')"
  if [[ -z "$secret_value" ]]; then
    printf 'Secret %s/%s is missing session-secret. Fix it before deploying.\n' "$namespace" "$secret_name" >&2
    exit 1
  fi
  secret_length="$(printf '%s' "$secret_value" | openssl base64 -d -A | wc -c | tr -d '[:space:]')"
  if (( secret_length < 32 )); then
    printf 'Secret %s/%s has a session-secret shorter than 32 bytes. Replace it before deploying.\n' "$namespace" "$secret_name" >&2
    exit 1
  fi
else
  generated_secret="$(openssl rand -hex 32)"
  printf '%s' "$generated_secret" | kubectl -n "$namespace" create secret generic "$secret_name" --from-file=session-secret=/dev/stdin
fi

kubectl apply -f "$manifest_dir"
