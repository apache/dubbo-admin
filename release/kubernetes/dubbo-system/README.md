# Kubernetes deployment

From the repository root, run:

```sh
./release/kubernetes/dubbo-system/deploy.sh
```

The script creates the `dubbo-system` namespace if needed, generates a random
`session-secret` in the `dubbo-admin-auth` Kubernetes Secret on first install,
and applies the manifests in this directory. It keeps the existing Secret on
later runs, so the signing key remains stable across upgrades.

This command requires `kubectl` access to create the namespace, Secret, and
manifest resources, plus `openssl` for the initial key generation. To apply the
manifests directly with `kubectl apply -f`, create `dubbo-admin-auth` with a
`session-secret` key in the `dubbo-system` namespace first.
