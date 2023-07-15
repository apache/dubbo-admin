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
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/app/dubboctl/internal/util"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/apache/dubbo-admin/app/dubboctl/internal/manifest"
	"helm.sh/helm/v3/pkg/action"
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

var DefaultFilters = []util.FilterFunc{
	util.LicenseFilter,
	util.FormatterFilter,
	util.SpaceFilter,
}

// Renderer is responsible for rendering helm chart with new values.
// For using RenderManifest, we must invoke Init firstly.
type Renderer interface {
	Init() error
	RenderManifest(valsYaml string) (string, error)
}

type RendererOptions struct {
	Name      string
	Namespace string

	// fields for LocalRenderer
	// local file system containing the target chart
	FS fs.FS
	// chart relevant path to FS
	Dir string

	// fields for RemoteRenderer
	// remote chart version
	Version string
	// remote chart repo url
	RepoURL string
}

type RendererOption func(*RendererOptions)

func WithName(name string) RendererOption {
	return func(opts *RendererOptions) {
		opts.Name = name
	}
}

func WithNamespace(ns string) RendererOption {
	return func(opts *RendererOptions) {
		opts.Namespace = ns
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

func WithVersion(version string) RendererOption {
	return func(opts *RendererOptions) {
		opts.Version = version
	}
}

func WithRepoURL(repo string) RendererOption {
	return func(opts *RendererOptions) {
		opts.RepoURL = repo
	}
}

// LocalRenderer load chart from local file system
type LocalRenderer struct {
	Opts    *RendererOptions
	Chart   *chart.Chart
	Started bool
}

func (lr *LocalRenderer) Init() error {
	fileNames, err := getFileNames(lr.Opts.FS, lr.Opts.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("chart of bootstrap %s doesn't exist", lr.Opts.Name)
		}
		return fmt.Errorf("getFileNames err: %s", err)
	}
	var files []*loader.BufferedFile
	for _, fileName := range fileNames {
		data, err := fs.ReadFile(lr.Opts.FS, fileName)
		if err != nil {
			return fmt.Errorf("ReadFile %s err: %s", fileName, err)
		}
		// todo:// explain why we need to do this
		name := manifest.StripPrefix(fileName, lr.Opts.Dir)
		file := &loader.BufferedFile{
			Name: name,
			Data: data,
		}
		files = append(files, file)
	}
	newChart, err := loader.LoadFiles(files)
	if err != nil {
		return fmt.Errorf("load chart of bootstrap %s err: %s", lr.Opts.Name, err)
	}
	lr.Chart = newChart
	lr.Started = true
	return nil
}

func (lr *LocalRenderer) RenderManifest(valsYaml string) (string, error) {
	if !lr.Started {
		return "", errors.New("LocalRenderer has not been init")
	}
	return renderManifest(valsYaml, lr.Chart, true, lr.Opts, DefaultFilters...)
}

func NewLocalRenderer(opts ...RendererOption) (Renderer, error) {
	newOpts := &RendererOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}

	if err := verifyRendererOptions(newOpts); err != nil {
		return nil, fmt.Errorf("verify err: %s", err)
	}
	return &LocalRenderer{
		Opts: newOpts,
	}, nil
}

type RemoteRenderer struct {
	Opts    *RendererOptions
	Chart   *chart.Chart
	Started bool
}

func (rr *RemoteRenderer) initChartPathOptions() *action.ChartPathOptions {
	// for now, using RepoURL and Version directly
	return &action.ChartPathOptions{
		RepoURL: rr.Opts.RepoURL,
		Version: rr.Opts.Version,
	}
}

func (rr *RemoteRenderer) Init() error {
	cpOpts := rr.initChartPathOptions()
	settings := cli.New()
	// using release name as chart name by default
	cp, err := locateChart(cpOpts, rr.Opts.Name, settings)
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	if err := verifyInstallable(chartRequested); err != nil {
		return err
	}

	rr.Chart = chartRequested
	rr.Started = true

	return nil
}

func (rr *RemoteRenderer) RenderManifest(valsYaml string) (string, error) {
	if !rr.Started {
		return "", errors.New("RemoteRenderer has not been init")
	}
	return renderManifest(valsYaml, rr.Chart, false, rr.Opts, DefaultFilters...)
}

func NewRemoteRenderer(opts ...RendererOption) (Renderer, error) {
	newOpts := &RendererOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}

	//if err := verifyRendererOptions(newOpts); err != nil {
	//	return nil, NewLocalRendererErrMgr.WithDescF("verify err: %s", err)
	//}
	return &RemoteRenderer{
		Opts: newOpts,
	}, nil
}

func verifyRendererOptions(opts *RendererOptions) error {
	if opts.Name == "" {
		return errors.New("missing bootstrap name for Renderer")
	}
	if opts.Namespace == "" {
		// logger.Log("using default namespace)
		opts.Namespace = identifier.DubboSystemNamespace
	}
	if opts.FS == nil {
		return errors.New("missing chart FS for Renderer")
	}
	if opts.Dir == "" {
		return errors.New("missing chart dir for Renderer")
	}
	return nil
}

// read all files recursively under root path from a certain local file system
func getFileNames(f fs.FS, root string) ([]string, error) {
	var fileNames []string
	if err := fs.WalkDir(f, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fileNames = append(fileNames, path)
		return nil
	}); err != nil {
		return nil, err
	}
	return fileNames, nil
}

func verifyInstallable(cht *chart.Chart) error {
	typ := cht.Metadata.Type
	if typ == "" || typ == "application" {
		return nil
	}
	return fmt.Errorf("%s chart %s is not installable", typ, cht.Name())
}

func renderManifest(valsYaml string, cht *chart.Chart, builtIn bool, opts *RendererOptions, filters ...util.FilterFunc) (string, error) {
	valsMap := make(map[string]any)
	if err := yaml.Unmarshal([]byte(valsYaml), &valsMap); err != nil {
		return "", fmt.Errorf("unmarshal failed err: %s", err)
	}
	RelOpts := chartutil.ReleaseOptions{
		Name:      opts.Name,
		Namespace: opts.Namespace,
	}
	// todo:// need to specify k8s version
	caps := chartutil.DefaultCapabilities
	// maybe we need a configuration to change this caps
	resVals, err := chartutil.ToRenderValues(cht, valsMap, RelOpts, caps)
	if err != nil {
		return "", fmt.Errorf("ToRenderValues failed err: %s", err)
	}
	// todo: // explain why there is a hack way
	if builtIn {
		resVals["Values"].(chartutil.Values)["enabled"] = true
	}
	filesMap, err := engine.Render(cht, resVals)
	if err != nil {
		return "", fmt.Errorf("Render chart failed err: %s", err)
	}
	keys := make([]string, 0, len(filesMap))
	for key := range filesMap {
		// remove notation files such as Notes.txt
		if strings.HasSuffix(key, NotesFileNameSuffix) {
			continue
		}
		keys = append(keys, key)
	}
	// to ensure that every manifest rendered by same values are the same
	sort.Strings(keys)

	var builder strings.Builder
	for i := 0; i < len(keys); i++ {
		file := filesMap[keys[i]]
		file = util.ApplyFilters(file, filters...)
		// ignore empty manifest
		if file == "" {
			continue
		}
		if !strings.HasSuffix(file, YAMLSeparator) {
			file += YAMLSeparator
		}
		builder.WriteString(file)
	}

	return builder.String(), nil
}

// locateChart locate the target chart path by sequential orders:
// 1. find local helm repository using "name-version.tgz" format
// 2. using downloader to pull remote chart
func locateChart(cpOpts *action.ChartPathOptions, name string, settings *cli.EnvSettings) (string, error) {
	name = strings.TrimSpace(name)
	version := strings.TrimSpace(cpOpts.Version)

	// check if it's in Helm's chart cache
	// cacheName is hardcoded as format of helm. eg: grafana-6.31.1.tgz
	cacheName := name + "-" + cpOpts.Version + ".tgz"
	cachePath := path.Join(settings.RepositoryCache, cacheName)
	if _, err := os.Stat(cachePath); err == nil {
		abs, err := filepath.Abs(cachePath)
		if err != nil {
			return abs, err
		}
		if cpOpts.Verify {
			if _, err := downloader.VerifyChart(abs, cpOpts.Keyring); err != nil {
				return "", err
			}
		}
		return abs, nil
	}

	dl := downloader.ChartDownloader{
		Out:     os.Stdout,
		Keyring: cpOpts.Keyring,
		Getters: getter.All(settings),
		Options: []getter.Option{
			getter.WithPassCredentialsAll(cpOpts.PassCredentialsAll),
			getter.WithTLSClientConfig(cpOpts.CertFile, cpOpts.KeyFile, cpOpts.CaFile),
			getter.WithInsecureSkipVerifyTLS(cpOpts.InsecureSkipTLSverify),
		},
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}

	if cpOpts.Verify {
		dl.Verify = downloader.VerifyAlways
	}
	if cpOpts.RepoURL != "" {
		chartURL, err := repo.FindChartInAuthAndTLSAndPassRepoURL(cpOpts.RepoURL, cpOpts.Username, cpOpts.Password, name, version,
			cpOpts.CertFile, cpOpts.KeyFile, cpOpts.CaFile, cpOpts.InsecureSkipTLSverify, cpOpts.PassCredentialsAll, getter.All(settings))
		if err != nil {
			return "", err
		}
		name = chartURL

		// Only pass the user/pass on when the user has said to or when the
		// location of the chart repo and the chart are the same domain.
		u1, err := url.Parse(cpOpts.RepoURL)
		if err != nil {
			return "", err
		}
		u2, err := url.Parse(chartURL)
		if err != nil {
			return "", err
		}

		// Host on URL (returned from url.Parse) contains the port if present.
		// This check ensures credentials are not passed between different
		// services on different ports.
		if cpOpts.PassCredentialsAll || (u1.Scheme == u2.Scheme && u1.Host == u2.Host) {
			dl.Options = append(dl.Options, getter.WithBasicAuth(cpOpts.Username, cpOpts.Password))
		} else {
			dl.Options = append(dl.Options, getter.WithBasicAuth("", ""))
		}
	} else {
		dl.Options = append(dl.Options, getter.WithBasicAuth(cpOpts.Username, cpOpts.Password))
	}

	// if RepositoryCache doesn't exist, create it
	if err := os.MkdirAll(settings.RepositoryCache, 0o755); err != nil {
		return "", err
	}

	filename, _, err := dl.DownloadTo(name, version, settings.RepositoryCache)
	if err != nil {
		return "", err
	}

	lname, err := filepath.Abs(filename)
	if err != nil {
		return filename, err
	}
	return lname, nil
}
