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
image="${DUBBO_ADMIN_IMAGE:-}"

if [[ -z "$image" ]]; then
  printf 'Set DUBBO_ADMIN_IMAGE to an image built with the session-secret fix; the bundled 0.7.0 image is vulnerable.\n' >&2
  exit 1
fi

if [[ "$image" == *@sha256:* ]]; then
  image_name="${image%@sha256:*}"
  image_digest="${image##*@sha256:}"
  if [[ ! "$image_name" =~ ^[A-Za-z0-9._:/-]+$ || ! "$image_digest" =~ ^[a-fA-F0-9]{64}$ ]]; then
    printf 'DUBBO_ADMIN_IMAGE must be a valid image reference with a sha256 digest.\n' >&2
    exit 1
  fi
  image_override="    digest: sha256:$image_digest"
else
  image_name="${image%:*}"
  image_tag="${image##*:}"
  if [[ "$image_name" == "$image" || ! "$image_name" =~ ^[A-Za-z0-9._:/-]+$ || ! "$image_tag" =~ ^[A-Za-z0-9_.-]+$ || "$image_tag" == 0.7.0 || "$image_tag" == latest ]]; then
    printf 'DUBBO_ADMIN_IMAGE must use an explicit fixed-image tag other than 0.7.0 or latest.\n' >&2
    exit 1
  fi
  image_override="    newTag: $image_tag"
fi

for command in kubectl openssl; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'Required command not found: %s\n' "$command" >&2
    exit 1
  fi
done

render_dir="$(mktemp -d)"
trap 'rm -rf "$render_dir"' EXIT
cp "$manifest_dir"/*.yaml "$render_dir"/
{
  printf 'apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n'
  for manifest in "$manifest_dir"/*.yaml; do
    printf '  - %s\n' "$(basename "$manifest")"
  done
  printf 'images:\n  - name: apache/dubbo-admin\n    newName: %s\n%s\n' "$image_name" "$image_override"
} > "$render_dir/kustomization.yaml"
rendered_manifest="$(kubectl kustomize "$render_dir")"

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

printf '%s\n' "$rendered_manifest" | kubectl apply -f -
