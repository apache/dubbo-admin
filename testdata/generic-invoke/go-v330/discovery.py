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
"""Authenticated real-discovery verifier. Cookies and credentials stay in memory."""
import argparse
import http.cookiejar
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request

parser = argparse.ArgumentParser()
parser.add_argument('--case', choices=['all', 'bad-signature', 'bad-args', 'unreachable', 'admin-unreachable'], default='all')
args = parser.parse_args()
base = os.environ.get('ADMIN_URL', 'http://127.0.0.1:8888').rstrip('/')
mesh = os.environ.get('MESH', 'generic-go')
service = 'example.GoEchoService'
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

def request(path, body=None, query=None, form=False):
    if query is not None:
        path += '?' + urllib.parse.urlencode(query)
    data = None if body is None else (urllib.parse.urlencode(body).encode() if form else json.dumps(body).encode())
    req = urllib.request.Request(base + path, data=data)
    if data is not None:
        req.add_header('Content-Type', 'application/x-www-form-urlencoded' if form else 'application/json')
    with opener.open(req, timeout=15) as response:
        return json.load(response)

def success(response):
    if response.get('code') != 'Success':
        raise RuntimeError('API failed: ' + str(response.get('code')))
    return response['data']

# Check that an actually unavailable HTTP service produces a nonzero exit.
# Port zero cannot be a listening TCP service. No external component is stopped.
if args.case == 'admin-unreachable':
    try:
        with opener.open('http://127.0.0.1:0/api/v1/auth/login', timeout=2):
            pass
    except urllib.error.URLError:
        raise SystemExit(42)
    raise SystemExit('unavailable service unexpectedly reachable')

# Wait for authentication endpoint and asynchronous metadata/instance discovery.
for attempt in range(90):
    try:
        success(request('/api/v1/auth/login', {'user': os.environ['ADMIN_USER'], 'password': os.environ['ADMIN_PASSWORD']}, form=True))
        break
    except (OSError, ValueError, RuntimeError):
        if attempt == 89:
            raise SystemExit('authentication unavailable or rejected')
        time.sleep(1)
query = {'mesh': mesh, 'serviceName': service}
for attempt in range(90):
    instances = success(request('/api/v1/service/provider-instances', query=query))
    methods = success(request('/api/v1/service/methods', query=query))
    if instances and methods:
        break
    time.sleep(1)
else:
    raise SystemExit('real discovery did not produce instances and signatures')
instance = instances[0]['name']

def resolve_method(method):
    # Go can expose both the exported name and its lower-case-first alias.
    # Match only the first letter case; reject overloads instead of guessing a signature.
    aliases = {method, method[:1].lower() + method[1:], method[:1].upper() + method[1:]}
    candidates = [m for m in methods if m['methodName'] in aliases]
    signatures = {m.get('signature') for m in candidates}
    if not candidates or None in signatures or '' in signatures or len(signatures) != 1:
        raise RuntimeError('missing/ambiguous real signature: ' + method)
    selected = next((m for m in candidates if m['methodName'] == method), candidates[0])
    name, sig = selected['methodName'], selected['signature']
    detail = success(request('/api/v1/service/method/detail', query=dict(query, methodName=name, signature=sig)))
    if detail.get('methodName') != name or detail.get('signature') != sig:
        raise RuntimeError('method detail name/signature mismatch')
    return name, sig

def invoke(method, values, timeout=3000, sig=None, target=None):
    name, discovered_signature = resolve_method(method)
    return request('/api/v1/service/generic/invoke', dict(query, instanceName=target or instance, methodName=name, signature=discovered_signature if sig is None else sig, args=values, timeoutMs=timeout, attachments={'traceId': 'go-e2e'}))

if args.case != 'all':
    if args.case == 'bad-signature':
        response = invoke('echo', [1], sig='int->java.lang.String')
        expected = 'NotFoundError'
    elif args.case == 'bad-args':
        response = invoke('echo', [])
        expected = 'InvalidArgument'
    else:
        # Target-resolution failure is deterministic and does not stop an external Provider.
        target = os.environ.get('UNREACHABLE_INSTANCE_NAME')
        response = invoke('echo', ['hello'], target=target or 'unreachable-go-e2e-instance')
        expected = 'InternalError' if target else 'NotFoundError'
    if response.get('code') != expected:
        raise SystemExit('negative case returned unexpected code')
    raise SystemExit(42)

for method, values, expected in [('echo', ['hello'], 'hello'), ('echoInt', [42], 42), ('echoNested', [{'item': {'value': 42}}], {'item': {'value': 42}}), ('noArgs', [], 'no-args'), ('echoArray', [[1, 2]], [1, 2]), ('attachments', [], 'go-e2e')]:
    data = success(invoke(method, values))
    if method == 'echoNested':
        result = data['rawResult']
        if not isinstance(result, dict) or not isinstance(result.get('item'), dict):
            raise SystemExit('nested result/item must be objects')
        value = result['item'].get('value')
        # bool is an int subclass in Python; exact type also rejects 42.0 and "42".
        if type(value) is not int or value != 42:
            raise SystemExit('nested value must be integer 42')
    if data['rawResult'] != expected or data['elapsedMs'] < 0:
        raise SystemExit('unexpected result: ' + method)
started = time.monotonic()
response = invoke('delay', [1000], timeout=100)
# Current implementation reports RPC timeouts as InternalError; classification is Task C.
if response.get('code') != 'InternalError' or time.monotonic() - started > 3:
    raise SystemExit('timeout observation failed')
print('Authenticated discovery, signature details, RPC values and timeout passed')
# TODO(#1631/#1647): overflow (including nested integer values), without rounded-value acceptance.
# TODO(#1632): business exception classification, without weakening transport assertions.
