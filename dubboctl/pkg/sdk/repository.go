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

package sdk

import (
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Repository struct {
	Name     string
	Runtimes []Runtime
	fs       util.Filesystem
	uri      string
}

type repositoryConfig struct {
	DefaultName   string `yaml:"name,omitempty"`
	TemplatesPath string `yaml:"templates,omitempty"`
}

func NewRepository(name, uri string) (r Repository, err error) {
	r = Repository{
		uri: uri,
	}
	fs, err := filesystemFromURI(uri)
	if err != nil {
		return Repository{}, fmt.Errorf("failed to get repository from URI (%q): %w", uri, err)
	}
	r.fs = fs
	repoConfig := repositoryConfig{}
	if repoConfig.TemplatesPath != "" {
		if err = checkDir(r.fs, repoConfig.TemplatesPath); err != nil {
			err = fmt.Errorf("templates path '%v' does not exist in repo '%v'. %v",
				repoConfig.TemplatesPath, r.Name, err)
			return
		}
	} else {
		repoConfig.TemplatesPath = "."
	}

	r.Name, err = repositoryDefaultName(repoConfig.DefaultName, uri)
	if err != nil {
		return
	}
	if name != "" {
		r.Name = name
	}
	r.Runtimes, err = repositoryRuntimes(fs, r.Name, repoConfig)
	return
}

func repositoryDefaultName(name, uri string) (string, error) {
	if name != "" {
		return name, nil
	}
	if uri != "" {
		parsed, err := url.Parse(uri)
		if err != nil {
			return "", err
		}
		ss := strings.Split(parsed.Path, "/")
		if len(ss) > 0 {
			return strings.TrimSuffix(ss[len(ss)-1], ".git"), nil
		}
	}
	return DefaultRepositoryName, nil
}

func repositoryRuntimes(fs util.Filesystem, repoName string, repoConfig repositoryConfig) (runtimes []Runtime, err error) {
	runtimes = []Runtime{}
	fis, err := fs.ReadDir(repoConfig.TemplatesPath)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		runtime := Runtime{
			Name: fi.Name(),
		}

		runtime.Templates, err = runtimeTemplates(fs, repoConfig.TemplatesPath, repoName, runtime.Name)
		if err != nil {
			return
		}
		runtimes = append(runtimes, runtime)
	}
	return
}

func runtimeTemplates(fs util.Filesystem, templatesPath, repoName, runtimeName string) (templates []Template, err error) {
	runtimePath := path.Join(templatesPath, runtimeName)
	if err = checkDir(fs, runtimePath); err != nil {
		err = fmt.Errorf("runtime path '%v' not found. %v", runtimePath, err)
		return
	}

	fis, err := fs.ReadDir(runtimePath)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		t := template{
			name:       fi.Name(),
			repository: repoName,
			runtime:    runtimeName,
			fs:         util.NewSubFS(path.Join(runtimePath, fi.Name()), fs),
		}
		templates = append(templates, t)
	}
	return
}

type Runtime struct {
	Name      string
	Templates []Template
}

func (r *Repository) Template(runtimeName, name string) (t Template, err error) {
	runtime, err := r.Runtime(runtimeName)
	if err != nil {
		return
	}
	for _, t := range runtime.Templates {
		if t.Name() == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template not found")
}

func (r *Repository) Templates(runtimeName string) ([]Template, error) {
	for _, runtime := range r.Runtimes {
		if runtime.Name == runtimeName {
			return runtime.Templates, nil
		}
	}
	return nil, nil
}

func (r *Repository) Runtime(name string) (runtime Runtime, err error) {
	if name == "" {
		return Runtime{}, fmt.Errorf("language runtime required")
	}
	for _, runtime = range r.Runtimes {
		if runtime.Name == name {
			return runtime, err
		}
	}
	return Runtime{}, fmt.Errorf("language runtime not found")
}

func (r *Repository) Write(dest string) (err error) {
	if r.fs == nil {
		return errors.New("the write operation is not supported on this repo")
	}
	fs := r.fs

	if _, ok := r.fs.(util.BillyFilesystem); ok {
		var (
			tempDir string
			clone   *git.Repository
			wt      *git.Worktree
		)
		if tempDir, err = os.MkdirTemp("", "dubbo"); err != nil {
			return
		}
		if clone, err = git.PlainClone(tempDir, false,
			getGitCloneOptions(r.uri)); err != nil {
			return fmt.Errorf("failed to plain clone repository: %w", err)
		}
		if wt, err = clone.Worktree(); err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
		fs = util.NewBillyFilesystem(wt.Filesystem)
	}
	return util.CopyFromFS(".", dest, fs)
}

func (r *Repository) URL() string {
	uri := r.uri

	if uri == "" {
		return ""
	}

	if strings.HasPrefix(uri, "file://") {
		uri = filepath.FromSlash(r.uri[7:])
	}

	repo, err := git.PlainOpen(uri)
	if err != nil {
		return ""
	}

	c, err := repo.Config()
	if err != nil {
		return ""
	}

	ref, _ := repo.Head()
	if _, ok := c.Remotes["origin"]; ok {
		urls := c.Remotes["origin"].URLs
		if len(urls) > 0 {
			return urls[0] + "#" + ref.Name().Short()
		}
	}
	return ""
}

func filesystemFromURI(uri string) (fs util.Filesystem, err error) {
	if uri == "" {
		return EmbeddedTemplatesFS, nil
	}
	if isNonBareGitRepo(uri) {
		return filesystemFromPath(uri)
	}

	fs, err = FilesystemFromRepo(uri)
	if fs != nil || err != nil {
		return
	}

	return filesystemFromPath(uri)
}

func isNonBareGitRepo(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	if parsed.Scheme != "file" {
		return false
	}
	p := filepath.Join(filepath.FromSlash(uri[7:]), ".git")
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func filesystemFromPath(uri string) (fs util.Filesystem, err error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return
	}

	if parsed.Scheme != "file" {
		return nil, fmt.Errorf("only file scheme is supported")
	}

	path := filepath.FromSlash(uri[7:])

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %v", path)
	}
	return util.NewOsFilesystem(path), nil
}

func FilesystemFromRepo(uri string) (util.Filesystem, error) {
	clone, err := git.Clone(memory.NewStorage(),
		memfs.New(),
		getGitCloneOptions(uri),
	)
	if err != nil {
		if isRepoNotFoundError(err) {
			return nil, nil
		}

		if isBranchNotFoundError(err) {
			return nil, fmt.Errorf("failed to clone repository: branch not found for uri %s", uri)
		}
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	wt, err := clone.Worktree()
	if err != nil {
		return nil, err
	}
	return util.NewBillyFilesystem(wt.Filesystem), nil
}

func getGitCloneOptions(uri string) *git.CloneOptions {
	branch := ""
	splitUri := strings.Split(uri, "#")
	if len(splitUri) > 1 {
		uri = splitUri[0]
		branch = splitUri[1]
	}

	opt := &git.CloneOptions{
		URL: uri, Depth: 1, Tags: git.NoTags,
		RecurseSubmodules: git.NoRecurseSubmodules,
	}
	if branch != "" {
		opt.ReferenceName = plumbing.NewBranchReferenceName(branch)
	}
	return opt
}

func isRepoNotFoundError(err error) bool {
	return err != nil && err.Error() == "repository not found"
}

func isBranchNotFoundError(err error) bool {
	return err != nil && err.Error() == "reference not found"
}

func checkDir(fs util.Filesystem, path string) error {
	fi, err := fs.Stat(path)
	if err != nil && os.IsNotExist(err) {
		err = fmt.Errorf("path '%v` not found", path)
	} else if err == nil && !fi.IsDir() {
		err = fmt.Errorf("path '%v' is not a directory", path)
	}
	return err
}
