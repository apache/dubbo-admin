# Go 3.3.0 generic invocation fixture

This is an independent Go module pinned to `dubbo.apache.org/dubbo-go/v3 v3.3.0`. No root module replace or dependency change is needed. It exposes the same `example.GoEchoService` over Dubbo/Hessian2 and Triple unary/Hessian2. Go 1.23 or newer, Docker (only for an optional isolated ZooKeeper), and Python 3 (L3) are needed.

```sh
bash testdata/generic-invoke/go-v330/zookeeper.sh start
cd testdata/generic-invoke/go-v330
go build -mod=readonly ./...
go run -mod=readonly .
```

Configuration (all local defaults bind/connect to loopback):

| Variable | Default | Meaning |
| --- | --- | --- |
| PROVIDER_HOST | 127.0.0.1 | RPC bind/advertised IP; use reachable pod IP in Kubernetes |
| DUBBO_PORT | 20880 | Dubbo TCP listener |
| TRIPLE_PORT | 50051 | Triple unary listener |
| ZOOKEEPER_ADDR | 127.0.0.1:2181 | ZooKeeper host:port, without scheme |
| OBSERVATION_ADDR | 127.0.0.1:18090 | HTTP observation listener; use 0.0.0.0:18090 only inside an isolated container |
| ZOOKEEPER_PORT | 2181 | Host port used by optional ZooKeeper script |
| ZK_RECEIPT | system temporary directory | Exact Docker container-ID receipt; use the same value for start/stop |

Methods use Java-compatible lower-case aliases: `echo(string)`, `echoInt(int32)`, `echoNested(example.Envelope)` with `item.value` declared `int32`, `noArgs()`, `echoArray([]int32)`, `attachments()` returning the inbound `traceId`, and `delay(delayMs int32)` (0..10000). Remote metadata is reported to ZooKeeper. Obtain actual wire signatures through Admin discovery, rather than inventing them. `GET /count` returns `{"count":N}`; all business methods increment the same counter exactly once, observations never increment it. Use an isolated fixture with no parallel callers when checking rejected-request counts. Do not publish observation ports publicly.

PowerShell equivalents (save this ID for cleanup; no named/shared container is deleted):

```powershell
$id = docker run -d --label generic-invoke-fixture=true -p 127.0.0.1:2181:2181 zookeeper:3.8
if ($LASTEXITCODE -ne 0) { throw 'ZooKeeper start failed' }
$env:ZOOKEEPER_ADDR = '127.0.0.1:2181'
Push-Location testdata/generic-invoke/go-v330
try { go run -mod=readonly . } finally { Pop-Location }
# After stopping this Provider with Ctrl+C:
if ($id -notmatch '^[a-f0-9]{64}$') { throw 'Invalid container receipt' }
if ((docker inspect -f '{{index .Config.Labels "generic-invoke-fixture"}}' $id) -ne 'true') { throw 'Ownership mismatch' }
docker rm -f $id
```

Shell cleanup: stop your Provider process, then `bash testdata/generic-invoke/go-v330/zookeeper.sh stop`. Scripts never start Docker from `go test`. See [the invocation guide](../../../../docs/generic-invoke.md) for L2/L3, authentication and cleanup.
