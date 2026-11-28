# Generic invocation: Go Provider verification

The fixture in [go-v330](../testdata/generic-invoke/go-v330/README.md) is an independent module pinned to the released dubbo-go v3.3.0. It provides real RPC checks and authenticated discovery verification against an isolated Go Provider.

## L2: real RPC with deterministic catalog fixtures

Start the isolated ZooKeeper and Provider as described in the fixture README. The service test uses existing `genericServiceE2EContext` / `genericServiceE2ERouter`; the handler test directly uses `genericHTTPTestContext` / `genericHTTPTestRouter`. Both seed memory Instance, ServiceProviderMetadata and RPCInstance resources following the Java test structure. These are L2 tests: catalog fixtures are in memory, RPC transport is real, and the RPC function is never replaced.

```sh
go test -mod=readonly ./pkg/console/service ./pkg/console/handler
RUN_DUBBO_GO_E2E=1 go test -mod=readonly -p 1 -count=1 -v -run 'GenericInvoke.*GoProvider' ./pkg/console/service ./pkg/console/handler
```

The first command skips Go E2E by default and starts no containers. Once enabled, unreachable Provider/observation endpoints fail the tests. Run packages sequentially (`-p 1`) because count assertions require an isolated Provider. Environment overrides: `GO_PROVIDER_DUBBO_ADDR=127.0.0.1:20880`, `GO_PROVIDER_TRIPLE_ADDR=127.0.0.1:50051`, `GO_PROVIDER_OBSERVATION_URL=http://127.0.0.1:18090`. These also work with loopback Kubernetes port-forwards. Neither test changes metadata to hide a failing protocol.

Both layers assert Dubbo and Triple string/integer/nested-integer/no-argument/array values, inbound attachments, signature rejection with unchanged Provider count, and timeout with an observed Provider request. Large-integer overflow, nested-integer overflow, and fine-grained business exception classification are deferred in the current suite and marked with TODO placeholders in the tests.

PowerShell:

```powershell
$env:RUN_DUBBO_GO_E2E='1'
$env:GO_PROVIDER_DUBBO_ADDR='127.0.0.1:20880'
$env:GO_PROVIDER_TRIPLE_ADDR='127.0.0.1:50051'
$env:GO_PROVIDER_OBSERVATION_URL='http://127.0.0.1:18090'
go test -mod=readonly -p 1 -count=1 -v -run 'GenericInvoke.*GoProvider' ./pkg/console/service ./pkg/console/handler
Remove-Item Env:RUN_DUBBO_GO_E2E
```

## L3: authenticated real discovery

Admin must discover the same isolated ZooKeeper used by the Provider. Configure a zookeeper discovery with ID `generic-go`, registry `zookeeper://127.0.0.1:2181` (or the isolated service DNS inside a cluster), memory store, and mock engine for standalone discovery. The engine is independent of registry discovery; no cluster RBAC is required. Use an independently built Admin from the reviewed current branch and a deployment-specific session secret of at least 32 bytes. Supply your own user/password at runtime; never commit credentials or deployment-specific configuration.

For an already running Admin/Provider:

```sh
export ADMIN_URL=http://127.0.0.1:8888
export ZOOKEEPER_ADDR=127.0.0.1:2181
export MESH=generic-go
export ADMIN_USER=<runtime-user>
read -rsp 'Admin password: ' ADMIN_PASSWORD; echo; export ADMIN_PASSWORD
SKIP_LOCAL_START=1 bash testdata/generic-invoke/run-discovery-e2e.sh
unset ADMIN_PASSWORD
```

`ZOOKEEPER_ADDR` configures locally launched Provider/Admin; external mode never reconfigures an existing Admin. `MESH` must equal its discovery ID. The Provider's advertised IP must be reachable from the Admin process. L2 port-forward overrides do not rewrite the L3 discovered target.

For local startup, leave `SKIP_LOCAL_START` unset and set `ADMIN_BIN` to the absolute path of your freshly built branch Admin binary. The runner starts isolated ZooKeeper, compiles/starts the independent Provider and starts the supplied Admin at port 8888, writes runtime configuration/logs only in a private system temporary directory, then stops only its own processes/container and removes that directory on exit. Go/Docker may require network access for initial dependencies/images. Keep custom `ZOOKEEPER_PORT` and `ZOOKEEPER_ADDR` consistent. `PYTHON` defaults to `python3` and can point to a Python 3 executable. When the script owns local startup, use default `ADMIN_URL` or a URL pointing to that same local port 8888.

The verifier sends `POST /api/v1/auth/login` as form fields `user` and `password`, retains the session cookie only in memory, polls real provider instances and method signatures for up to 90 seconds, fetches method detail and invokes the API. No cookie, token, config or logs are written to the repository. Normal API failures exit nonzero. Negative cases deliberately return exit 42 only for the exact expected API error; the driver rejects any other exit code.

## HTTP sequence

Requests require the authenticated session cookie. The following PowerShell equivalent keeps cookies in memory; run from the repository root. The bundled Python verifier automates these steps with stronger assertions and all method cases.

```powershell
$base='http://127.0.0.1:8888'
$credential=Get-Credential -Message 'Isolated test Admin'
$login=Invoke-RestMethod -Method Post "$base/api/v1/auth/login" -Body @{user=$credential.UserName;password=$credential.GetNetworkCredential().Password} -SessionVariable session
if($login.code -ne 'Success'){throw 'Login failed'}
$q='mesh=generic-go&serviceName=example.GoEchoService'
$instances=Invoke-RestMethod "$base/api/v1/service/provider-instances?$q" -WebSession $session
$methods=Invoke-RestMethod "$base/api/v1/service/methods?$q" -WebSession $session
if($instances.code -ne 'Success' -or @($instances.data).Count -eq 0){throw 'Wait for real discovery'}
$method=@($methods.data | Where-Object methodName -eq 'echoArray')[0]
$sig=[uri]::EscapeDataString($method.signature)
$detail=Invoke-RestMethod "$base/api/v1/service/method/detail?$q&methodName=echoArray&signature=$sig" -WebSession $session
$body=@{mesh='generic-go';serviceName='example.GoEchoService';instanceName=$instances.data[0].name;methodName='echoArray';signature=$method.signature;args=@(,@(1,2));timeoutMs=3000;attachments=@{traceId='go-e2e'}} | ConvertTo-Json -Depth 10
$result=Invoke-RestMethod -Method Post "$base/api/v1/service/generic/invoke" -WebSession $session -ContentType application/json -Body $body
if($result.code -ne 'Success'){throw $result.code}
# Or run the complete L3 verifier with the same runtime credentials:
$env:ADMIN_URL=$base; $env:ADMIN_USER=$credential.UserName
$env:ADMIN_PASSWORD=$credential.GetNetworkCredential().Password
try {
 python testdata/generic-invoke/go-v330/discovery.py
 if($LASTEXITCODE -ne 0){throw 'L3 failed'}
 foreach($case in @('bad-signature','bad-args','unreachable','admin-unreachable')) {
  python testdata/generic-invoke/go-v330/discovery.py --case $case
  if($LASTEXITCODE -ne 42){throw "Negative self-check failed: $case"}
 }
} finally {Remove-Item Env:ADMIN_PASSWORD; $credential=$null; $session=$null}
```

The invoke body has `mesh`, `serviceName`, `instanceName`, `methodName`, the discovered `signature`, `args`, `timeoutMs` and `attachments`. No arguments use `args: []`; one string uses `args: ["hello"]`; one array uses `args: [[1,2]]` (the outer array is the argument list). Never flatten the array into two arguments. `attachments()` must return `go-e2e`. `delay` accepts `[1000]`; a 100 ms timeout must fail. A nested echo input is `[{"item":{"value":42}}]`; the integer field is declared in the Provider contract. Overflow rejection is not yet enforced by this suite.

Success is HTTP 200 with `code: "Success"`, `data.rawResult` and nonnegative `data.elapsedMs`. Business API errors may also use HTTP 200: always inspect `code`. An unknown signature/target yields `NotFoundError`; argument count/type validation yields `InvalidArgument`; current RPC failure/timeout yields `InternalError` with a generic message. Authentication failure is HTTP 401. Do not infer business exception classification from `InternalError`.

The deterministic `unreachable` negative case selects a nonexistent instance, testing target-resolution failure without stopping an externally owned Provider. `admin-unreachable` also verifies an actual connection failure against a closed loopback HTTP port and requires exit 42. To verify an actual RPC transport outage, set `UNREACHABLE_INSTANCE_NAME` to a discovered, unavailable Provider instance: the `unreachable` case then requires `InternalError` instead of `NotFoundError`. Do not claim that the missing-instance check proves RPC transport failure. L2 enabled tests independently fail fast on real unreachable RPC listeners. Cluster transport-outage injection is out of scope for this automated check.

## Cleanup

The local bash runner cleans its own temporary files, child processes and exact Docker container ID, including on failure. It leaves externally running components untouched. Manually started fixtures must be stopped explicitly as documented in the Provider README. Stop only the port-forward PIDs belonging to your test session. Cluster rebuild/cleanup scripts and kubeconfig belong outside this repository; cluster cleanup must use the recorded names, labels and UIDs, never a prefix-only delete.
