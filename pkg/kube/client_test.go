// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"context"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	inputPath = "./testdata/input"
	wantPath  = "./testdata/want"
)

func TestCtlClient_ApplyManifest(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "ctl_client-apply_manifest.yaml",
			want:  "ctl_client-apply_manifest.yaml",
		},
	}

	for _, test := range tests {
		var fakeCli client.Client
		ctlCli := CtlClient{Client: fakeCli}
		inputManifest, err := readManifest(test.input)
		if err != nil {
			t.Fatalf("read input manifest %s err: %s", test.input, err)
		}
		testNs := "test"
		if err := ctlCli.ApplyManifest(inputManifest, testNs); err != nil {
			t.Errorf("ApplyManifest failed, err: %s", err)
			return
		}
		nsKey := client.ObjectKey{
			Namespace: metav1.NamespaceSystem,
			Name:      testNs,
		}
		receiver := &corev1.Namespace{}
		if err := ctlCli.Get(context.Background(), nsKey, receiver); err != nil {
			t.Errorf("createNamespace failed, err: %s", err)
		}
		assert.Equal(t, testNs, receiver.Name)
	}
}

func TestCtlClient_ApplyObject(t *testing.T) {
	tests := []struct {
		desc string
		// object stated represented by manifest yaml before apply object
		before  string
		input   string
		want    string
		wantErr bool
	}{
		{
			desc:  "create object when this object is not found",
			input: "ctl_client-apply_object-create.yaml",
			want:  "ctl_client-apply_object-create.yaml",
		},
		{
			desc:   "update object when this object exists",
			before: "ctl_client-apply_object-update-before.yaml",
			input:  "ctl_client-apply_object-update.yaml",
			want:   "ctl_client-apply_object-update.yaml",
		},
	}
	for _, test := range tests {
		var fakeCli client.Client
		if test.before != "" {
			beforeManifest, err := readManifest(path.Join(inputPath, test.before))
			if err != nil {
				t.Fatalf("read before manifest %s err: %s", test.before, err)
			}
			beforeObj, err := ParseObjectFromManifest(beforeManifest)
			if err != nil {
				t.Fatalf("initialize object state failed, err: %s", err)
			}
			fakeCli = fake.NewClientBuilder().WithRuntimeObjects(beforeObj.Unstructured()).Build()
		} else {
			fakeCli = fake.NewClientBuilder().Build()
		}
		ctlCli := &CtlClient{Client: fakeCli}

		inputManifest, err := readManifest(path.Join(inputPath, test.input))
		if err != nil {
			t.Fatalf("read input manifest %s err: %s", test.input, err)
		}
		inputObj, err := ParseObjectFromManifest(inputManifest)
		if err != nil {
			t.Fatalf("parse input object failed, err: %s", err)
		}
		if err := ctlCli.ApplyObject(inputObj.Unstructured()); err != nil {
			t.Errorf("ApplyObject failed, err: %s", err)
			return
		}

		wantManifest, err := readManifest(path.Join(wantPath, test.want))
		if err != nil {
			t.Fatalf("read want manifest %s err: %s", test.want, err)
		}
		wantObj, err := ParseObjectFromManifest(wantManifest)
		if err != nil {
			t.Fatalf("parse want object failed, err: %s", err)
		}
		wantKey := client.ObjectKeyFromObject(wantObj.Unstructured())

		gotObj := inputObj
		if err := ctlCli.Get(context.Background(), wantKey, gotObj.Unstructured()); err != nil {
			t.Fatalf("get object failed, err: %s", err)
		}
		// remove additional fields added by k8s for testing
		unstructured.RemoveNestedField(gotObj.Unstructured().Object, "metadata", "resourceVersion")
		unstructured.RemoveNestedField(gotObj.Unstructured().Object, "metadata", "creationTimestamp")
		if !wantObj.IsEqual(gotObj) {
			// todo:// need to print the difference of wantObj and gotObj
			t.Error("gotObj and wantObj are not the same")
		}
	}
}

func readManifest(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
