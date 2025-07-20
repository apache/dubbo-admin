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

package helm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/manifests"
	"github.com/apache/dubbo-admin/operator/pkg/manifest"
	"github.com/apache/dubbo-admin/operator/pkg/util"
	"github.com/apache/dubbo-admin/operator/pkg/values"
	"github.com/apache/dubbo-admin/pkg/common/util/slices"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

const (
	// NotesFileNameSuffix is the file name suffix for helm notes.
	// see https://helm.sh/docs/chart_template_guide/notes_files/
	NotesFileNameSuffix = ".txt"
	BaseChartName       = "base"
)

type Warnings = util.Errors

// Render produces a set of fully rendered manifests from Helm.
// Any warnings are also propagated up.
// Note: this is the direct result of the Helm call. Postprocessing steps are done later.
func Render(namespace string, directory string, dop values.Map) ([]manifest.Manifest, util.Errors, error) {
	val, ok := dop.GetPathMap("spec.values")
	if !ok {
		return nil, nil, fmt.Errorf("failed to get values from dop: %v", ok)
	}
	path := pathJoin("charts", directory)
	f := manifests.BuiltinDir("")
	chrt, err := loadChart(f, path)
	output, warnings, err := renderChart(namespace, val, chrt)
	if err != nil {
		return nil, nil, fmt.Errorf("render chart: %v", err)
	}
	mfs, err := manifest.Parse(output)
	return mfs, warnings, err
}

// renderChart renders the given chart with the given values and returns the resulting YAML manifest string.
func renderChart(namespace string, chrtVals values.Map, chrt *chart.Chart) ([]string, Warnings, error) {
	opts := chartutil.ReleaseOptions{
		Name:      "dubbo",
		Namespace: namespace,
	}
	caps := *chartutil.DefaultCapabilities
	helmVals, err := chartutil.ToRenderValues(chrt, chrtVals, opts, &caps)
	if err != nil {
		return nil, nil, fmt.Errorf("converting values: %v", err)
	}
	files, err := engine.Render(chrt, helmVals)
	if err != nil {
		return nil, nil, err
	}
	crdFiles := chrt.CRDObjects()
	if chrt.Metadata.Name == BaseChartName {
		values.GetPathAs[bool](chrtVals, "base.enableDubboConfigCRD")
	}
	var warnings Warnings
	keys := make([]string, 0, len(files))
	for k := range files {
		if strings.HasSuffix(k, NotesFileNameSuffix) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := make([]string, 0, len(keys))
	for _, k := range keys {
		res = append(res, util.SplitString(files[k])...)
	}
	slices.SortBy(crdFiles, func(a chart.CRD) string {
		return a.Name
	})
	for _, crd := range crdFiles {
		res = append(res, util.SplitString(string(crd.File.Data))...)
	}
	return res, warnings, nil
}

// loadChart reads a chart from the filesystem. This is like loader.LoadDir but allows a fs.FS.
func loadChart(f fs.FS, root string) (*chart.Chart, error) {
	fnames, err := getFilesRecursive(f, root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("component does not exist")
		}
		return nil, fmt.Errorf("list files: %v", err)
	}
	var bfs []*loader.BufferedFile
	for _, fn := range fnames {
		b, err := fs.ReadFile(f, fn)
		if err != nil {
			return nil, fmt.Errorf("read file:%v", err)
		}
		name := strings.ReplaceAll(stripPrefix(fn, root), string(filepath.Separator), "/")
		bf := &loader.BufferedFile{
			Name: name,
			Data: b,
		}
		bfs = append(bfs, bf)
	}
	return loader.LoadFiles(bfs)
}

// stripPrefix removes the given prefix from prefix.
func stripPrefix(path, prefix string) string {
	pl := len(strings.Split(prefix, "/"))
	pv := strings.Split(path, "/")
	return strings.Join(pv[pl:], "/")
}

func getFilesRecursive(f fs.FS, root string) ([]string, error) {
	result := []string{}
	err := fs.WalkDir(f, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		result = append(result, path)
		return nil
	})
	return result, err
}
