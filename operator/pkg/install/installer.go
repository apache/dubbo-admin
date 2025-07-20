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
	"sync"
	"time"

	"github.com/apache/dubbo-admin/operator/pkg/component"
	"github.com/apache/dubbo-admin/operator/pkg/manifest"
	"github.com/apache/dubbo-admin/operator/pkg/uninstall"
	"github.com/apache/dubbo-admin/operator/pkg/util"
	"github.com/apache/dubbo-admin/operator/pkg/util/clog"
	"github.com/apache/dubbo-admin/operator/pkg/util/dmultierr"
	"github.com/apache/dubbo-admin/operator/pkg/util/progress"
	"github.com/apache/dubbo-admin/operator/pkg/values"
	"github.com/apache/dubbo-admin/pkg/common/util/sets"
	"github.com/apache/dubbo-admin/pkg/common/util/slices"
	"github.com/apache/dubbo-admin/pkg/kube"
	"github.com/hashicorp/go-multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
)

type Installer struct {
	DryRun       bool
	SkipWait     bool
	Kube         kube.CLIClient
	Values       values.Map
	ProgressInfo *progress.Log
	WaitTimeout  time.Duration
	Logger       clog.Logger
}

// InstallManifests applies a set of rendered manifests to the cluster.
func (i Installer) InstallManifests(manifests []manifest.ManifestSet) error {
	err := i.installSystemNamespace()
	if err != nil {
		return err
	}
	if err := i.install(manifests); err != nil {
		return err
	}
	return nil
}

// installSystemNamespace creates the system namespace before installation.
func (i Installer) installSystemNamespace() error {
	ns := i.Values.GetPathStringOr("metadata.namespace", "dubbo-system")
	if err := util.CreateNamespace(i.Kube.Kube(), ns, i.DryRun); err != nil {
		return err
	}
	return nil
}

// install takes rendered manifests and actually applies them to the cluster.
// This considers ordering based on components.
func (i Installer) install(manifests []manifest.ManifestSet) error {
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := dmultierr.New()
	if err := errors.ErrorOrNil(); err != nil {
		return err
	}

	disabledComponents := sets.New(slices.Map(component.AllComponents, func(e component.Component) component.Name {
		return e.UserFacingName
	})...)
	dependencyWaitCh := dependenciesChannels()
	for _, mfs := range manifests {
		mfs := mfs
		c := mfs.Components
		m := mfs.Manifests
		disabledComponents.Delete(c)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s := dependencyWaitCh[c]; s != nil {
				<-s
			}
			if len(m) != 0 {
				if err := i.applyManifestSet(mfs); err != nil {
					mu.Lock()
					errors = multierror.Append(errors, err)
					mu.Unlock()
				}
			}
			for _, ch := range componentDependencies[c] {
				dependencyWaitCh[ch] <- struct{}{}
			}

		}()
	}
	for cc := range disabledComponents {
		for _, ch := range componentDependencies[cc] {
			dependencyWaitCh[ch] <- struct{}{}
		}
	}
	wg.Wait()
	if err := errors.ErrorOrNil(); err != nil {
		return err
	}

	if err := i.prune(manifests); err != nil {
		return fmt.Errorf("pruning: %v", err)
	}
	i.ProgressInfo.SetState(progress.StateComplete)
	return nil
}

// applyManifestSet applies a set of manifests to the cluster.
func (i Installer) applyManifestSet(manifestSet manifest.ManifestSet) error {
	componentNames := string(manifestSet.Components)
	manifests := manifestSet.Manifests
	pi := i.ProgressInfo.NewComponent(componentNames)
	for _, obj := range manifests {
		obj, err := i.applyLabelsAndAnnotations(obj, componentNames)
		if err != nil {
			return err
		}
		if err := i.serverSideApply(obj); err != nil {
			pi.ReportError(err.Error())
			return err
		}
		pi.ReportProgress()
	}
	if err := WaitForResources(manifests, i.Kube, i.WaitTimeout, i.DryRun, pi); err != nil {
		werr := fmt.Errorf("failed to wait for resource: %v", err)
		pi.ReportError(werr.Error())
		return werr
	}
	pi.ReportFinished()
	return nil
}

// serverSideApply creates or updates an object in the API server depending on whether it already exists.
func (i Installer) serverSideApply(obj manifest.Manifest) error {
	const fieldManager = "dubbo-operator"
	dc, err := i.Kube.DynamicClientFor(obj.GroupVersionKind(), obj.Unstructured, "")
	if err != nil {
		return err
	}
	objStr := fmt.Sprintf("%s/%s/%s", obj.GetKind(), obj.GetNamespace(), obj.GetName())

	var dryRun []string
	if i.DryRun {
		return nil
	}
	if _, err := dc.Patch(context.TODO(), obj.GetName(), types.ApplyPatchType, []byte(obj.Content), metav1.PatchOptions{
		DryRun:       dryRun,
		FieldManager: fieldManager,
	}); err != nil {
		return fmt.Errorf("failed to update resource with server-side apply for obj %v: %v", objStr, err)
	}
	return nil
}

func (i Installer) applyLabelsAndAnnotations(obj manifest.Manifest, cname string) (manifest.Manifest, error) {
	for k, v := range getOwnerLabels(i.Values, cname) {
		err := util.SetLabel(obj, k, v)
		if err != nil {
			return manifest.Manifest{}, err
		}
	}
	// We're mutating the unstructured, must rebuild the YAML.
	return manifest.FromObject(obj.Unstructured)
}

// prune removes resources that are in the cluster, but not a part of the currently installed set of objects.
func (i Installer) prune(manifests []manifest.ManifestSet) error {
	if i.DryRun {
		return nil
	}

	i.ProgressInfo.SetState(progress.StatePruning)

	// Build up a map of component->resources, so we know what to keep around
	excluded := map[component.Name]sets.String{}
	// Include all components in case we disabled some.
	for _, c := range component.AllComponents {
		excluded[c.UserFacingName] = sets.New[string]()
	}
	for _, mfs := range manifests {
		for _, m := range mfs.Manifests {
			excluded[mfs.Components].Insert(m.Hash())
		}
	}

	coreLabels := getOwnerLabels(i.Values, "")
	selector := klabels.Set(coreLabels).AsSelectorPreValidated()
	compReq, err := klabels.NewRequirement(manifest.DubboComponentLabel, selection.Exists, nil)
	if err != nil {
		return err
	}
	selector = selector.Add(*compReq)

	var errs util.Errors
	resources := uninstall.PrunedResourcesSchemas()
	for _, gvk := range resources {
		dc, err := i.Kube.DynamicClientFor(gvk, nil, "")
		if err != nil {
			return err
		}
		objs, err := dc.List(context.Background(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if objs == nil {
			continue
		}
		for comp, excluded := range excluded {
			compLabels := klabels.SelectorFromSet(getOwnerLabels(i.Values, string(comp)))
			for _, obj := range objs.Items {
				if excluded.Contains(manifest.ObjectHash(&obj)) {
					continue
				}
				if obj.GetLabels()[manifest.OwningResourceNotPruned] == "true" {
					continue
				}
				// Label mismatch. Provided objects don't select against the component, so this likely means the object
				// is for another component.
				if !compLabels.Matches(klabels.Set(obj.GetLabels())) {
					continue
				}
				if err := uninstall.DeleteResource(i.Kube, i.DryRun, i.Logger, &obj); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errs.ToError()
}

var componentDependencies = map[component.Name][]component.Name{
	component.BaseComponentName: {
		component.NacosRegisterComponentName,
		component.ZookeeperRegisterComponentName,
		component.AdminComponentName,
	},
	component.NacosRegisterComponentName:     {},
	component.ZookeeperRegisterComponentName: {},
	component.AdminComponentName:             {},
}

func dependenciesChannels() map[component.Name]chan struct{} {
	ret := make(map[component.Name]chan struct{})
	for _, parent := range componentDependencies {
		for _, child := range parent {
			ret[child] = make(chan struct{}, 1)
		}
	}
	return ret
}
func getOwnerLabels(dop values.Map, c string) map[string]string {
	labels := make(map[string]string)

	if n := dop.GetPathString("metadata.name"); n != "" {
		labels[manifest.OwningResourceName] = n
	}
	if n := dop.GetPathString("metadata.namespace"); n != "" {
		labels[manifest.OwningResourceNamespace] = n
	}

	if c != "" {
		labels[manifest.DubboComponentLabel] = c
	}
	return labels
}
