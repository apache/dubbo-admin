/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package uninstall

import (
	"context"
	"fmt"

	"github.com/apache/dubbo-admin/operator/pkg/manifest"
	"github.com/apache/dubbo-admin/operator/pkg/util"
	"github.com/apache/dubbo-admin/operator/pkg/util/clog"
	"github.com/apache/dubbo-admin/operator/pkg/util/ptr"
	"github.com/apache/dubbo-admin/pkg/config/schema/gvk"
	"github.com/apache/dubbo-admin/pkg/kube"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
)

var (
	// ClusterResources are resource types the operator prunes, ordered by which types should be deleted, first to last.
	ClusterResources = []schema.GroupVersionKind{
		gvk.MutatingWebhookConfiguration.Kubernetes(),
		gvk.ValidatingWebhookConfiguration.Kubernetes(),
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
	}
	// ClusterControlPlaneResources lists cluster scope resources types which should be deleted during uninstall command.
	ClusterControlPlaneResources = []schema.GroupVersionKind{
		gvk.MutatingWebhookConfiguration.Kubernetes(),
		gvk.ValidatingWebhookConfiguration.Kubernetes(),
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
	}
	// AllClusterResources lists all cluster scope resources types which should be deleted in purge case, including CRD.
	AllClusterResources = append(ClusterResources,
		gvk.CustomResourceDefinition.Kubernetes(),
	)
)

// GetRemovedResources get the list of resources to be removed
// 1. if includeClusterResources is false, we list the namespaced resources by matching revision and component labels.
// 2. if includeClusterResources is true, we list the namespaced and cluster resources by component labels only.
// If componentName is not empty, only resources associated with specific components would be returned
// UnstructuredList of objects and corresponding list of name kind hash of k8sObjects would be returned
func GetRemovedResources(kc kube.CLIClient, dopName, dopNamespace string, includeClusterResources bool) ([]*unstructured.UnstructuredList, error) {
	var usList []*unstructured.UnstructuredList
	labels := make(map[string]string)
	if dopName != "" {
		labels[manifest.OwningResourceName] = dopName
	}
	if dopNamespace != "" {
		labels[manifest.OwningResourceNamespace] = dopNamespace
	}
	resources := NamespacedResources()
	gvkList := append(resources, ClusterControlPlaneResources...)
	if includeClusterResources {
		gvkList = append(resources, AllClusterResources...)
	}
	for _, g := range gvkList {
		var result *unstructured.UnstructuredList
		compReq, err := klabels.NewRequirement(manifest.DubboComponentLabel, selection.Exists, nil)
		if err != nil {
			return nil, err
		}
		c, err := kc.DynamicClientFor(g, nil, "")
		if err != nil {
			return nil, err
		}
		if includeClusterResources {
			s := klabels.NewSelector()
			result, err = c.List(context.Background(), metav1.ListOptions{LabelSelector: s.Add(*compReq).String()})
		}
		if result == nil || len(result.Items) == 0 {
			continue
		}
		usList = append(usList, result)

	}
	return usList, nil
}

// NamespacedResources gets specific pruning resources based on the k8s version
func NamespacedResources() []schema.GroupVersionKind {
	res := []schema.GroupVersionKind{
		gvk.Deployment.Kubernetes(),
		gvk.StatefulSet.Kubernetes(),
		gvk.DaemonSet.Kubernetes(),
		gvk.Service.Kubernetes(),
		gvk.ConfigMap.Kubernetes(),
		gvk.Secret.Kubernetes(),
		gvk.ServiceAccount.Kubernetes(),
		gvk.Job.Kubernetes(),
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
		{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"},
	}
	return res
}

func PrunedResourcesSchemas() []schema.GroupVersionKind {
	return append(NamespacedResources(), ClusterResources...)
}

// DeleteObjectsList removed resources that are in the slice of UnstructuredList.
func DeleteObjectsList(c kube.CLIClient, dryRun bool, log clog.Logger, objectsList []*unstructured.UnstructuredList) error {
	var errs util.Errors
	for _, ol := range objectsList {
		for _, o := range ol.Items {
			if err := DeleteResource(c, dryRun, log, &o); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.ToError()
}

func DeleteResource(kc kube.CLIClient, dryRun bool, log clog.Logger, obj *unstructured.Unstructured) error {
	name := fmt.Sprintf("%v/%s.%s", obj.GroupVersionKind(), obj.GetName(), obj.GetNamespace())
	if dryRun {
		fmt.Printf("Not remove object %s because of dry run.", name)
		return nil
	}

	c, err := kc.DynamicClientFor(obj.GroupVersionKind(), obj, "")
	if err != nil {
		return err
	}

	if err := c.Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{PropagationPolicy: ptr.Of(metav1.DeletePropagationForeground)}); err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		// do not return error if resources are not found
		log.LogAndPrintf("object: %s is not being deleted because it no longer exists", name)

		return nil
	}

	log.LogAndPrintf("✔︎ Removed %s.", name)

	return nil
}
