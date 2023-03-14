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

package render

import (
	"errors"
	"io/fs"
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/manifest"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"sigs.k8s.io/yaml"
)

const (
	YAMLSeparator       = "\n---\n"
	NotesFileNameSuffix = ".txt"
)

type Renderer interface {
	Init() error
	RenderManifest(valsYaml string) (string, error)
}

type RendererOptions struct {
	Name      string
	NameSpace string
	FS        fs.FS
	Dir       string
}

type RendererOption func(*RendererOptions)

func WithName(name string) RendererOption {
	return func(opts *RendererOptions) {
		opts.Name = name
	}
}

func WithNameSpace(ns string) RendererOption {
	return func(opts *RendererOptions) {
		opts.NameSpace = ns
	}
}

func WithFS(f fs.FS) RendererOption {
	return func(opts *RendererOptions) {
		opts.FS = f
	}
}

func WithDir(dir string) RendererOption {
	return func(opts *RendererOptions) {
		opts.Dir = dir
	}
}

type LocalRenderer struct {
	Opts    *RendererOptions
	Chart   *chart.Chart
	Started bool
}

func (lr *LocalRenderer) Init() error {
	fileNames, err := manifest.GetFileNames(lr.Opts.FS, lr.Opts.Dir)
	if err != nil {
		return err
	}
	var files []*loader.BufferedFile
	for _, fileName := range fileNames {
		data, err := fs.ReadFile(lr.Opts.FS, fileName)
		if err != nil {
			return err
		}
		name := manifest.StripPrefix(fileName, lr.Opts.Dir)
		file := &loader.BufferedFile{
			Name: name,
			Data: data,
		}
		files = append(files, file)
	}
	newChart, err := loader.LoadFiles(files)
	if err != nil {
		return err
	}
	lr.Chart = newChart
	lr.Started = true
	return nil
}

func (lr *LocalRenderer) RenderManifest(valsYaml string) (string, error) {
	if !lr.Started {
		return "", errors.New("LocalRender has not been init")
	}
	valsMap := make(map[string]any)
	if err := yaml.Unmarshal([]byte(valsYaml), &valsMap); err != nil {
		return "", err
	}
	RelOpts := chartutil.ReleaseOptions{
		Name:      "dubbo",
		Namespace: lr.Opts.NameSpace,
	}
	caps := chartutil.DefaultCapabilities
	// maybe we need a configuration to change this caps
	resVals, err := chartutil.ToRenderValues(lr.Chart, valsMap, RelOpts, caps)
	if err != nil {
		return "", err
	}
	resVals["Values"].(chartutil.Values)["enabled"] = true
	filesMap, err := engine.Render(lr.Chart, resVals)
	if err != nil {
		return "", err
	}
	keys := make([]string, 0, len(filesMap))
	for key := range filesMap {
		if strings.HasSuffix(key, NotesFileNameSuffix) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for i := 0; i < len(keys); i++ {
		file := filesMap[keys[i]]
		file = strings.TrimSpace(file) + "\n"
		if !strings.HasSuffix(file, YAMLSeparator) {
			file += YAMLSeparator
		}
		if _, err := builder.WriteString(file); err != nil {
			return "", err
		}
	}

	return builder.String(), nil
}

func NewLocalRenderer(opts ...RendererOption) (Renderer, error) {
	newOpts := &RendererOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// verify
	return &LocalRenderer{
		Opts: newOpts,
	}, nil
}
