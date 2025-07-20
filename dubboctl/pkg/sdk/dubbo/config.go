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

package dubbo

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DataDir         = ".dubbo"
	LogFile         = ".dubbo/dubbo.log"
	Dockerfile      = "Dockerfile"
	DefaultTemplate = "common"
	builtLogFile    = "built.log"
	built           = "built"
)

type DubboConfig struct {
	Root        string     `yaml:"-"`
	Name        string     `yaml:"name,omitempty" jsonschema:"pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"`
	Image       string     `yaml:"image,omitempty"`
	ImageDigest string     `yaml:"-"`
	Runtime     string     `yaml:"runtime,omitempty"`
	Template    string     `yaml:"template,omitempty"`
	Created     time.Time  `yaml:"created,omitempty"`
	Build       BuildSpec  `yaml:"build,omitempty"`
	Deploy      DeploySpec `yaml:"deploy,omitempty"`
}

type BuildSpec struct {
	BuilderImages map[string]string `yaml:"builderImages,omitempty"`
	Buildpacks    []string          `yaml:"buildpacks,omitempty"`
}

type DeploySpec struct {
	Namespace  string `yaml:"namespace,omitempty"`
	Output     string `yaml:"output,omitempty"`
	Port       int    `yaml:"port,omitempty"`
	TargetPort int    `yaml:"targetPort,omitempty"`
	NodePort   int    `yaml:"nodePort,omitempty"`
}

func NewDubboConfig(path string) (*DubboConfig, error) {
	var err error

	f := &DubboConfig{}
	f.Build.BuilderImages = make(map[string]string)

	if path == "" {
		if path, err = os.Getwd(); err != nil {
			return f, err
		}
	}
	f.Root = path

	fd, err := os.Stat(path)
	if err != nil {
		return f, err
	}
	if !fd.IsDir() {
		return nil, fmt.Errorf("function path must be a directory")
	}

	filename := filepath.Join(path, LogFile)
	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return f, err
	}

	bb, err := os.ReadFile(filename)
	if err != nil {
		return f, err
	}
	err = yaml.Unmarshal(bb, f)
	if err != nil {
		return f, err
	}

	return f, nil
}

func NewDubboConfigWithTemplate(dc *DubboConfig, initialized bool) *DubboConfig {
	if !initialized {
		if dc.Template == "" {
			dc.Template = DefaultTemplate
		}
		if dc.Template == "" {
			dc.Template = "initialized"
		}
	}
	if dc.Build.BuilderImages == nil {
		dc.Build.BuilderImages = make(map[string]string)
	}
	return dc
}

func (dc *DubboConfig) WriteFile() (err error) {
	file := filepath.Join(dc.Root, LogFile)
	var bytes []byte
	if bytes, err = yaml.Marshal(dc); err != nil {
		return
	}
	if err = os.WriteFile(file, bytes, 0o644); err != nil {
		return
	}
	return
}

func (dc *DubboConfig) WriteDockerfile(cmd *cobra.Command) (err error) {
	path := filepath.Join(dc.Root, Dockerfile)
	bytes, ok := DockerfileByRuntime[dc.Runtime]
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "The runtime of your current project is not one of Java or go. We cannot help you generate a Dockerfile template.")
		return
	}
	if err = os.WriteFile(path, []byte(bytes), 0o644); err != nil {
		return
	}
	return
}

func (dc *DubboConfig) Validate() error {
	if dc.Root == "" {
		return errors.New("dubbo root path is required")
	}

	var ctr int
	errs := [][]string{
		validateOptions(),
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("'%v' contains errors:", LogFile))

	for _, ee := range errs {
		if len(ee) > 0 {
			b.WriteString("\n") // Precede each group of errors with a linebreak
		}
		for _, e := range ee {
			ctr++
			b.WriteString("\t" + e)
		}
	}

	if ctr == 0 {
		return nil // Return nil if there were no validation errors.
	}

	return errors.New(b.String())
}

func (dc *DubboConfig) Built() bool {
	stamp := dc.buildStamp()
	if stamp == "" {
		return false
	}

	if dc.Image == "" {
		return false
	}

	hash, _, err := Fingerprint(dc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error calculating function's fingerprint: %v\n", err)
		return false
	}
	return stamp == hash
}

func (dc *DubboConfig) buildStamp() string {
	path := filepath.Join(dc.Root, DataDir, built)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

type (
	stampOptions struct {
		log bool
	}
	stampOption func(o *stampOptions)
)

func (dc *DubboConfig) Stamp(oo ...stampOption) (err error) {
	options := &stampOptions{}
	for _, o := range oo {
		o(options)
	}
	if err = runDataDir(dc.Root); err != nil {
		return
	}

	var hash, log string
	if hash, log, err = Fingerprint(dc); err != nil {
		return
	}

	if err = os.WriteFile(filepath.Join(dc.Root, DataDir, built), []byte(hash), os.ModePerm); err != nil {
		return err
	}

	blt := builtLogFile
	if options.log {
		blt = timestamp(blt)
	}
	logfile, err := os.Create(filepath.Join(dc.Root, DataDir, blt))
	if err != nil {
		return
	}
	defer logfile.Close()
	_, err = fmt.Fprintln(logfile, log)
	return
}

func timestamp(s string) string {
	t := time.Now()
	return fmt.Sprintf("%s.%09d.%s", t.Format("20060102150405"), t.Nanosecond(), s)
}

func (dc *DubboConfig) Initialized() bool {
	return !dc.Created.IsZero()
}

func Fingerprint(dc *DubboConfig) (hash, log string, err error) {
	h := sha256.New()   // Hash builder
	l := bytes.Buffer{} // Log buffer

	root := dc.Root
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	// TODO
	output := ""

	err = filepath.Walk(abs, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if info.IsDir() && (info.Name() == DataDir || info.Name() == ".git" || info.Name() == ".idea") {
			return filepath.SkipDir
		}
		if info.Name() == LogFile || info.Name() == Dockerfile || info.Name() == output {
			return nil
		}
		fmt.Fprintf(h, "%v:%v:", path, info.ModTime().UnixNano())   // Write to the Hashed
		fmt.Fprintf(&l, "%v:%v\n", path, info.ModTime().UnixNano()) // Write to the Log
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil)), l.String(), err
}

func runDataDir(root string) error {
	if err := os.MkdirAll(filepath.Join(root, DataDir), os.ModePerm); err != nil {
		return err
	}
	filePath := filepath.Join(root, ".gitignore")
	roFile, err := os.Open(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer roFile.Close()
	if !os.IsNotExist(err) {
		s := bufio.NewScanner(roFile)
		for s.Scan() {
			if strings.HasPrefix(s.Text(), "# /"+DataDir) { // if it was commented
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "#/"+DataDir) {
				return nil // user wants it
			}
			if strings.HasPrefix(s.Text(), "/"+DataDir) { // if it is there
				return nil // we're done
			}
		}
	}
	roFile.Close()
	rwFile, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer rwFile.Close()
	if _, err = rwFile.WriteString(`
# Applications use the .dubbo directory for local runtime data which should
# generally not be tracked in source control. To instruct the system to track
# .dubbo in source control, comment the following line (prefix it with '# ').
/.dubbo
`); err != nil {
		return err
	}
	if err = rwFile.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: error when syncing .gitignore. %s\n", err)
	}
	return nil
}

func validateOptions() []string {
	return nil
}
