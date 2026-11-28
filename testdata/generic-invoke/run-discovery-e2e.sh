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
umask 077
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
export ADMIN_URL=${ADMIN_URL:-http://127.0.0.1:8888}
export ZOOKEEPER_ADDR=${ZOOKEEPER_ADDR:-127.0.0.1:2181}
export MESH=${MESH:-generic-go}
: "${ADMIN_USER:?set ADMIN_USER}"
: "${ADMIN_PASSWORD:?set ADMIN_PASSWORD in the process environment}"
python=${PYTHON:-python3}
work=$(mktemp -d)
pids=()
cleanup() {
 trap - EXIT INT TERM
 for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; done
 if [[ ${started_zk:-0} == 1 ]]; then bash "$root/testdata/generic-invoke/go-v330/zookeeper.sh" stop; fi
 rm -rf -- "$work"
}
trap cleanup EXIT
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM
export ZK_RECEIPT="$work/zk.id"
if [[ ${SKIP_LOCAL_START:-0} != 1 ]]; then
 : "${ADMIN_BIN:?set ADMIN_BIN to the freshly built branch Admin binary}"
 ADMIN_BIN=$(cd "$(dirname "$ADMIN_BIN")" && pwd)/$(basename "$ADMIN_BIN")
 bash "$root/testdata/generic-invoke/go-v330/zookeeper.sh" start
 started_zk=1
 # Give ZooKeeper a bounded readiness window; Python retries the TCP connection.
 "$python" - <<'PY'
import os,socket,time
host,port=os.environ['ZOOKEEPER_ADDR'].rsplit(':',1)
for attempt in range(60):
 try:
  with socket.create_connection((host,int(port)),1): break
 except OSError: time.sleep(1)
else: raise SystemExit('ZooKeeper unavailable')
PY
 (cd "$root/testdata/generic-invoke/go-v330" && go build -mod=readonly -o "$work/provider" .)
 (cd "$work" && exec ./provider >provider.log 2>&1) & pids+=("$!")
 export LOCAL_ADMIN_CONFIG="$work/admin.json"
 "$python" - <<'PY'
import os,json,secrets
cfg={'log':{'level':'info','outputPath':os.path.join(os.path.dirname(os.environ['LOCAL_ADMIN_CONFIG']),'admin.log')},'console':{'ginMode':'release','port':8888,'auth':{'user':os.environ['ADMIN_USER'],'password':os.environ['ADMIN_PASSWORD'],'sessionSecret':secrets.token_urlsafe(32),'expirationTime':3600}},'store':{'type':'memory'},'discovery':[{'type':'zookeeper','id':os.environ['MESH'],'name':'isolated-go','address':{'registry':'zookeeper://'+os.environ['ZOOKEEPER_ADDR']}}],'engine':{'type':'mock','id':'mock','name':'mock'}}
with open(os.environ['LOCAL_ADMIN_CONFIG'],'w') as f: json.dump(cfg,f)
PY
 (cd "$work" && exec "$ADMIN_BIN" run --config-file "$LOCAL_ADMIN_CONFIG" >admin.stdout 2>&1) & pids+=("$!")
fi
"$python" "$root/testdata/generic-invoke/go-v330/discovery.py"
# The same command used by callers must exit nonzero on invalid arguments and unavailable targets.
for negative in bad-signature bad-args unreachable admin-unreachable; do
 set +e
 "$python" "$root/testdata/generic-invoke/go-v330/discovery.py" --case "$negative"
 rc=$?
 set -e
 [[ $rc == 42 ]] || { echo "negative self-check $negative: expected 42, got $rc" >&2; exit 1; }
done
echo 'Discovery E2E and negative exit checks passed'
