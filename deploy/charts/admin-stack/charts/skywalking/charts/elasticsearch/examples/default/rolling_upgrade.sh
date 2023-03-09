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

#!/usr/bin/env bash -x

kubectl proxy || true &

make &
PROC_ID=$!

while kill -0 "$PROC_ID" >/dev/null 2>&1; do
    echo "PROCESS IS RUNNING"
    if curl --fail 'http://localhost:8001/api/v1/proxy/namespaces/default/services/elasticsearch-master:9200/_search' ; then
        echo "cluster is healthy"
    else
        echo "cluster not healthy!"
        exit 1
    fi
    sleep 1
done
echo "PROCESS TERMINATED"
exit 0
