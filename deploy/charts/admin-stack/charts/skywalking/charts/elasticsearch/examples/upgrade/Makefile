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

CHART := elasticsearch
RELEASE := helm-es-upgrade
FROM := 7.17.1	# upgrade from versions before 7.17.1 isn't compatible with 8.x

install:
	../../../helpers/upgrade.sh --chart $(CHART) --release $(RELEASE) --from $(FROM)
	# Rolling upgrade doesn't work when upgrading from clusters with security disabled.
	# This is because nodes with security enabled can't join a cluster with security disabled.
	# Every nodes need to be recreated at the same time so they can recreate a cluster with security enabled
	kubectl delete pod --selector=app=upgrade-master

test: install goss

purge:
	helm del $(RELEASE)
