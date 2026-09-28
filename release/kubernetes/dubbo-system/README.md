<!--
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# Kubernetes deployment

From the repository root, run:

```sh
./release/kubernetes/dubbo-system/deploy.sh
```

The script creates the `dubbo-system` namespace if needed, generates a random
`session-secret` in the `dubbo-admin-auth` Kubernetes Secret on first install,
and applies the manifests in this directory. It keeps the existing Secret on
later runs, so the signing key remains stable across upgrades. An existing
Secret without `session-secret`, or with a value shorter than 32 bytes, stops
deployment instead of silently changing the key.

This command requires `kubectl` access to create the namespace, Secret, and
manifest resources, plus `openssl` for the initial key generation. To apply the
manifests directly with `kubectl apply -f`, create `dubbo-admin-auth` with a
`session-secret` key in the `dubbo-system` namespace first.

The manifest currently references `apache/dubbo-admin:0.7.0`, which predates
the session-secret validation in this change. When publishing the fix, update
that image tag to a release built from the fixed source. Until then, use an
image built from this branch for deployment testing; merely injecting a Secret
into the old image does not fix its cookie-signing behavior.
