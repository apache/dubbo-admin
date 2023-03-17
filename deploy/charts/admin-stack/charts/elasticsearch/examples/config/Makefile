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

default: test

include ../../../helpers/examples.mk

RELEASE := helm-es-config
TIMEOUT := 1200s

install:
	helm upgrade --wait --timeout=$(TIMEOUT) --install --values values.yaml $(RELEASE) ../../

secrets:
	kubectl delete secret elastic-config-credentials elastic-config-secret elastic-config-slack elastic-config-custom-path || true
	kubectl create secret generic elastic-config-credentials --from-literal=password=changeme --from-literal=username=elastic
	kubectl create secret generic elastic-config-slack --from-literal=xpack.notification.slack.account.monitoring.secure_url='https://hooks.slack.com/services/asdasdasd/asdasdas/asdasd'
	kubectl create secret generic elastic-config-secret --from-file=xpack.watcher.encryption_key=./watcher_encryption_key
	kubectl create secret generic elastic-config-custom-path --from-literal=slack_url='https://hooks.slack.com/services/asdasdasd/asdasdas/asdasd' --from-literal=thing_i_don_tcare_about=test

test: secrets install goss

purge:
	helm del $(RELEASE)
