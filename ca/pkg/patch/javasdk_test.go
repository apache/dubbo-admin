// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package patch

import (
	"encoding/json"
	"github.com/apache/dubbo-admin/ca/pkg/config"
	"github.com/mattbaird/jsonpatch"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestName(t *testing.T) {
	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options)
	pod := &v1.Pod{}
	json.Unmarshal([]byte("{\"kind\":\"Pod\",\"apiVersion\":\"v1\",\"metadata\":{\"generateName\":\"nginx-ccc5c45fb-\",\"namespace\":\"default\",\"creationTimestamp\":null,\"labels\":{\"app\":\"nginx\",\"dubbo-ca.inject\":\"true\",\"pod-template-hash\":\"ccc5c45fb\"},\"annotations\":{\"kubernetes.io/psp\":\"ack.privileged\",\"redeploy-timestamp\":\"1677575493160\"},\"ownerReferences\":[{\"apiVersion\":\"apps/v1\",\"kind\":\"ReplicaSet\",\"name\":\"nginx-ccc5c45fb\",\"uid\":\"f3382cb7-dd97-48d2-a01a-1f75e019596d\",\"controller\":true,\"blockOwnerDeletion\":true}],\"managedFields\":[{\"manager\":\"kube-controller-manager\",\"operation\":\"Update\",\"apiVersion\":\"v1\",\"time\":\"2023-02-28T09:23:17Z\",\"fieldsType\":\"FieldsV1\",\"fieldsV1\":{\"f:metadata\":{\"f:annotations\":{\".\":{},\"f:redeploy-timestamp\":{}},\"f:generateName\":{},\"f:labels\":{\".\":{},\"f:app\":{},\"f:dubbo-ca.inject\":{},\"f:pod-template-hash\":{}},\"f:ownerReferences\":{\".\":{},\"k:{\\\"uid\\\":\\\"f3382cb7-dd97-48d2-a01a-1f75e019596d\\\"}\":{}}},\"f:spec\":{\"f:containers\":{\"k:{\\\"name\\\":\\\"nginx\\\"}\":{\".\":{},\"f:image\":{},\"f:imagePullPolicy\":{},\"f:name\":{},\"f:resources\":{\".\":{},\"f:requests\":{\".\":{},\"f:cpu\":{},\"f:memory\":{}}},\"f:terminationMessagePath\":{},\"f:terminationMessagePolicy\":{}}},\"f:dnsPolicy\":{},\"f:enableServiceLinks\":{},\"f:restartPolicy\":{},\"f:schedulerName\":{},\"f:securityContext\":{},\"f:terminationGracePeriodSeconds\":{}}}}]},\"spec\":{\"volumes\":[{\"name\":\"kube-api-access-h9hz2\",\"projected\":{\"sources\":[{\"serviceAccountToken\":{\"expirationSeconds\":3607,\"path\":\"token\"}},{\"configMap\":{\"name\":\"kube-root-ca.crt\",\"items\":[{\"key\":\"ca.crt\",\"path\":\"ca.crt\"}]}},{\"downwardAPI\":{\"items\":[{\"path\":\"namespace\",\"fieldRef\":{\"apiVersion\":\"v1\",\"fieldPath\":\"metadata.namespace\"}}]}}],\"defaultMode\":420}}],\"containers\":[{\"name\":\"nginx\",\"image\":\"nginx:latest\",\"resources\":{\"requests\":{\"cpu\":\"250m\",\"memory\":\"512Mi\"}},\"volumeMounts\":[{\"name\":\"kube-api-access-h9hz2\",\"readOnly\":true,\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\"}],\"terminationMessagePath\":\"/dev/termination-log\",\"terminationMessagePolicy\":\"File\",\"imagePullPolicy\":\"Always\"}],\"restartPolicy\":\"Always\",\"terminationGracePeriodSeconds\":30,\"dnsPolicy\":\"ClusterFirst\",\"serviceAccountName\":\"default\",\"serviceAccount\":\"default\",\"securityContext\":{},\"schedulerName\":\"default-scheduler\",\"tolerations\":[{\"key\":\"node.kubernetes.io/not-ready\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300},{\"key\":\"node.kubernetes.io/unreachable\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300}],\"priority\":0,\"enableServiceLinks\":true,\"preemptionPolicy\":\"PreemptLowerPriority\"},\"status\":{}}"), pod)

	origin, _ := json.Marshal(pod)
	newPod, _ := sdk.NewPod(pod)

	after, _ := json.Marshal(newPod)
	patch, _ := jsonpatch.CreatePatch(origin, after)
	n, _ := json.Marshal(patch)
	println(string(n))
}
