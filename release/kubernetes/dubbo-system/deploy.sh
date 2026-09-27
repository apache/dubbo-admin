#!/usr/bin/env bash
set -euo pipefail

namespace=dubbo-system
secret_name=dubbo-admin-auth
manifest_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! kubectl get namespace "$namespace" >/dev/null 2>&1; then
  kubectl create namespace "$namespace"
fi

if kubectl -n "$namespace" get secret "$secret_name" >/dev/null 2>&1; then
  secret_value="$(kubectl -n "$namespace" get secret "$secret_name" -o jsonpath='{.data.session-secret}')"
  if [[ -z "$secret_value" ]]; then
    printf 'Secret %s/%s is missing session-secret. Fix it before deploying.\n' "$namespace" "$secret_name" >&2
    exit 1
  fi
else
  openssl rand -base64 48 | kubectl -n "$namespace" create secret generic "$secret_name" --from-file=session-secret=/dev/stdin
fi

kubectl apply -f "$manifest_dir"
