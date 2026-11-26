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
# A receipt with the exact container ID prevents deletion of a same-name replacement.
action=${1:-start}
: "${ZK_RECEIPT:=${TMPDIR:-/tmp}/generic-go-zookeeper-${UID}.id}"
case "$action" in
 start)
  test ! -e "$ZK_RECEIPT" || { echo 'receipt already exists; stop that run first' >&2; exit 1; }
  id=$(docker run -d --label generic-invoke-fixture=true -p "127.0.0.1:${ZOOKEEPER_PORT:-2181}:2181" zookeeper:3.8)
  [[ "$id" =~ ^[a-f0-9]{64}$ ]] || { echo 'unexpected container id from docker run' >&2; exit 1; }
  printf '%s\n' "$id" > "$ZK_RECEIPT"
  ;;
 stop)
  if [[ ! -e "$ZK_RECEIPT" ]]; then
   exit 0
  fi
  if [[ ! -s "$ZK_RECEIPT" ]]; then
   rm -f -- "$ZK_RECEIPT"
   exit 0
  fi
  id=$(cat "$ZK_RECEIPT")
  [[ "$id" =~ ^[a-f0-9]{64}$ ]] || exit 1
  if ! label=$(docker inspect -f '{{index .Config.Labels "generic-invoke-fixture"}}' "$id" 2>/dev/null); then
   rm -f -- "$ZK_RECEIPT"
   exit 0
  fi
  test "$label" = true
  docker rm -f "$id" >/dev/null
  rm -f -- "$ZK_RECEIPT"
  ;;
 *) echo 'usage: zookeeper.sh start|stop' >&2; exit 2;;
esac
