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

The bundled `apache/dubbo-admin:0.7.0` image predates the session-secret fix.
Until a fixed official image is published, build this branch and make its image
available to your cluster. From the repository root, deploy it with:

```sh
DUBBO_ADMIN_IMAGE=<fixed-image:tag> ./release/kubernetes/dubbo-system/deploy.sh
```

The script requires an explicit image built from the fixed source, renders it
into the Deployment before applying anything, and rejects the known old
`0.7.0` tag and `latest`. It cannot verify the contents of a supplied image;
the operator must provide a fixed build.
It then creates the `dubbo-system` namespace if needed, generates a random
`session-secret` in the `dubbo-admin-auth` Kubernetes Secret on first install,
and applies the manifests. It keeps the existing Secret on later runs, so the
signing key remains stable across upgrades. An existing Secret without
`session-secret`, or with a value shorter than 32 bytes, stops deployment.

This command requires `kubectl` access to create the namespace, Secret, and
manifest resources, plus `openssl` for the initial key generation. An image
reference pinned by `@sha256:<digest>` is also accepted. Do not apply these
manifests directly with `kubectl apply -f`: that bypasses the image check and
uses the old image.

When publishing the fix, update the manifest and script to use a release built
from the fixed source by default. Merely injecting a Secret into the old image
does not change its cookie-signing behavior.
