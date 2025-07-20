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

package install

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/operator/pkg/manifest"
	"github.com/apache/dubbo-admin/operator/pkg/util/progress"
	"github.com/apache/dubbo-admin/pkg/config/schema/gvk"
	"github.com/apache/dubbo-admin/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kctldeployment "k8s.io/kubectl/pkg/util/deployment"
)

// deployment holds associated replicaSet for a deployment
type deployment struct {
	replicaSet *appsv1.ReplicaSet
	deployment *appsv1.Deployment
}

// WaitForResources polls to get the current status of various objects that are not immediately ready
// until all are ready or a timeout is reached.
func WaitForResources(objects []manifest.Manifest, client kube.CLIClient, waitTimeout time.Duration, dryRun bool, l *progress.ManifestLog) error {
	if dryRun {
		return nil
	}

	var notReady []string
	var debugInfo map[string]string

	// Check if we are ready immediately, to avoid the 2s delay below when we are already ready
	if ready, _, _, err := waitForResources(objects, client, l); err == nil && ready {
		return nil
	}

	errPoll := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, waitTimeout, false, func(context.Context) (bool, error) {
		isReady, notReadyObjects, debugInfoObjects, err := waitForResources(objects, client, l)
		notReady = notReadyObjects
		debugInfo = debugInfoObjects
		return isReady, err
	})

	messages := []string{}
	for _, id := range notReady {
		debug, f := debugInfo[id]
		if f {
			messages = append(messages, fmt.Sprintf("  %s (%s)", id, debug))
		} else {
			messages = append(messages, fmt.Sprintf("  %s", debug))
		}
	}
	if errPoll != nil {
		return fmt.Errorf("resources not ready after %v: %v\n%s", waitTimeout, errPoll, strings.Join(messages, "\n"))
	}
	return nil
}

func waitForResources(objects []manifest.Manifest, k kube.Client, l *progress.ManifestLog) (bool, []string, map[string]string, error) {
	pods := []corev1.Pod{}
	deployments := []deployment{}
	statefulsets := []*appsv1.StatefulSet{}
	namespaces := []corev1.Namespace{}
	crds := []apiextensions.CustomResourceDefinition{}

	for _, o := range objects {
		kind := o.GroupVersionKind().Kind
		switch kind {
		case gvk.CustomResourceDefinition.Kind:
			crd, err := k.Ext().ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), o.GetName(), metav1.GetOptions{})
			if err != nil {
				return false, nil, nil, err
			}
			crds = append(crds, *crd)

		case gvk.Namespace.Kind:
			namespace, err := k.Kube().CoreV1().Namespaces().Get(context.TODO(), o.GetName(), metav1.GetOptions{})
			if err != nil {
				return false, nil, nil, err
			}
			namespaces = append(namespaces, *namespace)
		case gvk.Deployment.Kind:
			currentDeployment, err := k.Kube().AppsV1().Deployments(o.GetNamespace()).Get(context.TODO(), o.GetName(), metav1.GetOptions{})
			if err != nil {
				return false, nil, nil, err
			}
			_, _, newReplicaSet, err := kctldeployment.GetAllReplicaSets(currentDeployment, k.Kube().AppsV1())
			if err != nil || newReplicaSet == nil {
				return false, nil, nil, err
			}
			newDeployment := deployment{
				newReplicaSet,
				currentDeployment,
			}
			deployments = append(deployments, newDeployment)
		case gvk.StatefulSet.Kind:
			sts, err := k.Kube().AppsV1().StatefulSets(o.GetNamespace()).Get(context.TODO(), o.GetName(), metav1.GetOptions{})
			if err != nil {
				return false, nil, nil, err
			}
			statefulsets = append(statefulsets, sts)
		}
	}

	resourceDebugInfo := map[string]string{}

	dr, dnr := deploymentsReady(k.Kube(), deployments, resourceDebugInfo)
	stsr, stsnr := statefulsetsReady(statefulsets)
	nsr, nnr := namespacesReady(namespaces)
	pr, pnr := podsReady(pods)
	crdr, crdnr := crdsReady(crds)
	isReady := dr && nsr && stsr && pr && crdr
	notReady := append(append(append(append(nnr, dnr...), pnr...), stsnr...), crdnr...)
	if !isReady {
		l.ReportWaiting(notReady)
	}
	return isReady, notReady, resourceDebugInfo, nil
}

func crdsReady(crds []apiextensions.CustomResourceDefinition) (bool, []string) {
	var notReady []string
	for _, crd := range crds {
		ready := false
		for _, cond := range crd.Status.Conditions {
			if cond.Type == apiextensions.Established && cond.Status == apiextensions.ConditionTrue {
				ready = true
				break
			}
		}
		if !ready {
			notReady = append(notReady, "dubbo", crd.Name)
		}
	}
	return len(notReady) == 0, notReady
}

func podsReady(pods []corev1.Pod) (bool, []string) {
	var notReady []string
	for _, pod := range pods {
		if !isPodReady(&pod) {
			notReady = append(notReady, "Pod/"+pod.Namespace+"/"+pod.Name)
		}
	}
	return len(notReady) == 0, notReady
}

func isPodReady(pod *corev1.Pod) bool {
	if len(pod.Status.Conditions) > 0 {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady &&
				condition.Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func namespacesReady(namespaces []corev1.Namespace) (bool, []string) {
	var notReady []string
	for _, namespace := range namespaces {
		if namespace.Status.Phase != corev1.NamespaceActive {
			notReady = append(notReady, "Namespace/"+namespace.Name)
		}
	}
	return len(notReady) == 0, notReady
}

func deploymentsReady(cs kubernetes.Interface, deployments []deployment, info map[string]string) (bool, []string) {
	var notReady []string
	for _, v := range deployments {
		if v.replicaSet.Status.ReadyReplicas >= *v.deployment.Spec.Replicas {
			// Ready
			continue
		}
		id := "Deployment/" + v.deployment.Namespace + "/" + v.deployment.Name
		notReady = append(notReady, id)
	}
	return len(notReady) == 0, notReady
}

func statefulsetsReady(statefulsets []*appsv1.StatefulSet) (bool, []string) {
	var notReady []string
	for _, sts := range statefulsets {
		if sts.Spec.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType &&
			sts.Status.UpdatedReplicas != sts.Status.Replicas {
			notReady = append(notReady, "StatefulSet/"+sts.Namespace+"/"+sts.Name)
		}
		if sts.Spec.UpdateStrategy.Type == appsv1.RollingUpdateStatefulSetStrategyType {
			var partition int
			replicas := 1
			if sts.Spec.UpdateStrategy.RollingUpdate != nil &&
				sts.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
				partition = int(*sts.Spec.UpdateStrategy.RollingUpdate.Partition)
			}
			if sts.Spec.Replicas != nil {
				replicas = int(*sts.Spec.Replicas)
			}
			expectedReplicas := replicas - partition
			if int(sts.Status.UpdatedReplicas) != expectedReplicas {
				notReady = append(notReady, "StatefulSet/"+sts.Namespace+"/"+sts.Name)
				continue
			}
		}
	}
	return len(notReady) == 0, notReady
}
