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

package render

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/apache/dubbo-admin/manifests"
	"github.com/apache/dubbo-admin/operator/cmd/validation"
	"github.com/apache/dubbo-admin/operator/pkg/apis"
	"github.com/apache/dubbo-admin/operator/pkg/component"
	"github.com/apache/dubbo-admin/operator/pkg/helm"
	"github.com/apache/dubbo-admin/operator/pkg/manifest"
	"github.com/apache/dubbo-admin/operator/pkg/util"
	"github.com/apache/dubbo-admin/operator/pkg/util/clog"
	"github.com/apache/dubbo-admin/operator/pkg/values"
	"github.com/apache/dubbo-admin/pkg/kube"
)

// MergeInputs merges the various configuration inputs into one single DubboOperator.
func MergeInputs(filenames []string, flags []string) (values.Map, error) {
	// We want our precedence order to be:
	// base < profile < auto-detected settings < files (in order) < --set flags (in order).
	// The tricky part is that we don't know where to read the profile from until we read the files/--set flags.
	// To handle this, we will first build up these values,
	// then apply them on top of the base once we determine which base to use.
	// Initial base values.
	ConfigBase, err := values.MapFromJSON([]byte(`{
	  "apiVersion": "install.dubbo.io/v1alpha1",
	  "kind": "DubboOperator",
	  "metadata": {},
	  "spec": {}
	}`))
	if err != nil {
		return nil, err
	}

	// Apply all passed in files
	for i, fn := range filenames {
		var b []byte
		var err error
		if fn == "-" {
			if i != len(filenames)-1 {
				return nil, fmt.Errorf("stdin is only allowed as the last filename")
			}
			b, err = io.ReadAll(os.Stdin)
		} else {
			b, err = os.ReadFile(strings.TrimSpace(fn))
		}

		if err := checkDops(string(b)); err != nil {
			return nil, fmt.Errorf("checkDops err:%v", err)
		}

		m, err := values.MapFromYAML(b)
		if err != nil {
			return nil, fmt.Errorf("yaml Unmarshal err:%v", err)
		}
		// Special hack to allow an empty spec to work. Should this be more generic?
		if m["spec"] == nil {
			delete(m, "spec")
		}
		ConfigBase.MergeFrom(m)
	}
	// Apply any --set flags
	if err := ConfigBase.SetSpecPaths(flags...); err != nil {
		return nil, err
	}

	path := ConfigBase.GetPathString("")
	profile := ConfigBase.GetPathString("spec.profile")
	// Now we have the base
	base, err := readProfile(path, profile)
	if err != nil {
		return base, err
	}
	// Merge the user values on top
	base.MergeFrom(ConfigBase)
	return base, nil
}

func checkDops(s string) error {
	mfs, err := manifest.ParseMultiple(s)
	if err != nil {
		return fmt.Errorf("unable to parse file: %v", err)
	}
	if len(mfs) > 1 {
		return fmt.Errorf("contains multiple DubboOperator CRs, only one per file is supported")
	}
	return nil
}

// readProfile reads a profile, from given path.
func readProfile(path, profile string) (values.Map, error) {
	if profile == "" {
		profile = "default"
	}
	base, err := readBuiltinProfile(path, "default")
	if err != nil {
		return nil, err
	}
	if profile == "default" {
		return base, nil
	}
	p, err := readBuiltinProfile(path, profile)
	if err != nil {
		return nil, err
	}
	base.MergeFrom(p)
	return base, nil
}

func readBuiltinProfile(path, profile string) (values.Map, error) {
	fs := manifests.BuiltinDir(path)
	file, err := fs.Open(fmt.Sprintf("profiles/%v.yaml", profile))
	if err != nil {
		return nil, fmt.Errorf("profile %q not found: %v", profile, err)
	}
	pb, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return values.MapFromYAML(pb)
}

// GenerateManifest produces fully rendered Kubernetes objects from rendering Helm charts.
// Inputs can be files and --set strings.
// Client is option; if it is provided, cluster-specific settings can be auto-detected.
// Logger is also option; if it is provided warning messages may be logged.
func GenerateManifest(files []string, setFlags []string, logger clog.Logger, _ kube.Client) ([]manifest.ManifestSet, values.Map, error) {
	// First, compute our final configuration input. This will be in the form of an DubboOperator, but as an unstructured values.Map.
	// This allows safe access to get/fetch values dynamically, and avoids issues are typing and whether we should emit empty fields.
	merged, err := MergeInputs(files, setFlags)
	if err != nil {
		return nil, nil, fmt.Errorf("merge inputs: %v", err)
	}
	// Validate the config. This can emit warnings to the logger. If force is set, errors will be logged as warnings but not returned.
	if err := validateDubboOperator(merged, logger); err != nil {
		return nil, nil, fmt.Errorf("validateDubboOperator err:%v", err)
	}
	var chartWarnings util.Errors
	allManifests := map[component.Name]manifest.ManifestSet{}
	for _, comp := range component.AllComponents {
		specs, err := comp.Get(merged)
		if err != nil {
			return nil, nil, fmt.Errorf("get component %v: %v", comp.UserFacingName, err)
		}
		for _, spec := range specs {
			// Each component may get a different view of the values; modify them as needed (with a copy)
			compVals := applyComponentValuesToHelmValues(comp, spec, merged)
			// Render the chart.
			rendered, warnings, err := helm.Render(spec.Namespace, comp.HelmSubDir, compVals)
			if err != nil {
				return nil, nil, fmt.Errorf("helm render: %v", err)
			}
			chartWarnings = util.AppendErrs(chartWarnings, warnings)
			// DubboOperator has a variety of processing steps that are done *after* Helm, such as patching. Apply any of these steps.
			finalized, err := postProcess(comp, rendered, compVals)
			if err != nil {
				return nil, nil, fmt.Errorf("post process: %v", err)
			}
			mfs, found := allManifests[comp.UserFacingName]
			if found {
				mfs.Manifests = append(mfs.Manifests, finalized...)
				allManifests[comp.UserFacingName] = mfs
			} else {
				allManifests[comp.UserFacingName] = manifest.ManifestSet{
					Components: comp.UserFacingName,
					Manifests:  finalized,
				}
			}
		}
	}

	// Log any warnings we got from the charts
	if logger != nil {
		for _, w := range chartWarnings {
			logger.LogAndErrorf("%s %v", "❗", w)
		}
	}

	val := make([]manifest.ManifestSet, 0, len(allManifests))
	for _, v := range allManifests {
		val = append(val, v)
	}
	return val, merged, nil
}

func validateDubboOperator(dop values.Map, logger clog.Logger) error {
	warnings, errs := validation.ParseAndValidateDubboOperator(dop)
	if err := errs.ToError(); err != nil {
		return err
	}
	if logger != nil {
		for _, w := range warnings {
			logger.LogAndErrorf("%s %v", "❗", w)
		}
	}
	return nil
}

// applyComponentValuesToHelmValues translates a generic values set into a component-specific one.
func applyComponentValuesToHelmValues(_ component.Component, spec apis.DefaultCompSpec, merged values.Map) values.Map {
	if spec.Namespace != "" {
		spec.Namespace = "dubbo-system"
	}
	return merged
}
