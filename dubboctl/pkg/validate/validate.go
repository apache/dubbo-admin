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

package validate

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	operator "github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	operatorvalidate "github.com/apache/dubbo-kubernetes/operator/pkg/apis/validation"
	"github.com/apache/dubbo-kubernetes/pkg/config/validation"
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"strings"
)

var (
	errFiles    = errors.New(`error: you must specify resources by -f.`)
	validFields = map[string]struct{}{
		"apiVersion": {},
		"kind":       {},
		"metadata":   {},
		"spec":       {},
		"status":     {},
	}
	fileExtensions = []string{".json", ".yaml", ".yml"}
)

type validator struct{}

func (v *validator) validateFile(path string, dubboNamespace *string, reader io.Reader, writer io.Writer) (validation.Warning, error) {
	yamlReader := yaml.NewYAMLReader(bufio.NewReader(reader))
	var errs error
	var warnings validation.Warning
	for {
		doc, err := yamlReader.Read()
		if err == io.EOF {
			return warnings, errs
		}
		if err != nil {
			errs = multierror.Append(errs, multierror.Prefix(err, fmt.Sprintf("failed to decode file %s: ", path)))
			return warnings, errs
		}
		if len(doc) == 0 {
			continue
		}
		out := map[string]any{}
		if err := yaml.UnmarshalStrict(doc, &out); err != nil {
			errs = multierror.Append(errs, multierror.Prefix(err, fmt.Sprintf("failed to decode file %s: ", path)))
			return warnings, errs
		}
		un := unstructured.Unstructured{Object: out}
		warning, err := v.validateResource(*dubboNamespace, &un, writer)
		if err != nil {
			errs = multierror.Append(errs, multierror.Prefix(err, fmt.Sprintf("%s/%s/%s:",
				un.GetKind(), un.GetNamespace(), un.GetName())))
		}
		if warning != nil {
			warnings = multierror.Append(warnings, multierror.Prefix(warning, fmt.Sprintf("%s/%s/%s:",
				un.GetKind(), un.GetNamespace(), un.GetName())))
		}
	}
}

func (v *validator) validateResource(dubboNamespace string, un *unstructured.Unstructured, writer io.Writer) (validation.Warning, error) {
	var errs error
	if errs != nil {
		return nil, errs
	}
	if un.GetAPIVersion() == operator.DubboOperatorGVK.GroupVersion().String() {
		if un.GetKind() == operator.DubboOperatorGVK.Kind {
			if err := checkFields(un); err != nil {
				return nil, err
			}
			warnings, err := operatorvalidate.ParseAndValidateDubboOperator(un.Object, nil)
			if err != nil {
				return nil, err
			}
			if len(warnings) > 0 {
				return validation.Warning(warnings.ToError()), nil
			}
		}
	}
	return nil, nil
}

func checkFields(un *unstructured.Unstructured) error {
	var errs error
	for key := range un.Object {
		if _, ok := validFields[key]; !ok {
			errs = multierror.Append(errs, fmt.Errorf("unknown field %q", key))
		}
	}
	return errs
}

func NewValidateCommand(ctx cli.Context) *cobra.Command {
	var files []string
	vc := &cobra.Command{
		Use:   "validate -f FILENAME [options]",
		Short: "Validate Dubbo rules files",
		Long:  "The validate command is used to validate the dubbo related rule file",
		Example: `  # Validate current deployments under 'default' namespace with in the cluster
  kubectl get deployments -o yaml | dubboctl validate -f -

  # Validate current services under 'default' namespace with in the cluster
  kubectl get services -o yaml | dubboctl validate -f -

  # Validate resource yaml specifications
  dubboctl validate -f default.yaml
`,

		Args:    cobra.NoArgs,
		Aliases: []string{"v"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			dn := ctx.Namespace()
			return validateFiles(&dn, files, cmd.OutOrStderr())
		},
	}
	flags := vc.PersistentFlags()
	flags.StringSliceVarP(&files, "filename", "f", nil, "Inputs of files to validate")
	return vc
}

func validateFiles(dubboNamespace *string, files []string, writer io.Writer) error {
	if len(files) == 0 {
		return errFiles
	}
	v := &validator{}
	var errs error
	var reader io.ReadCloser
	warningsByFilename := map[string]validation.Warning{}
	processFile := func(path string) {
		var err error
		if path == "-" {
			reader = io.NopCloser(os.Stdin)
		} else {
			reader, err = os.Open(path)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("cannot read file %q: %v", path, err))
				return
			}
		}
		warning, err := v.validateFile(path, dubboNamespace, reader, writer)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		err = reader.Close()
		if err != nil {
			fmt.Printf("file: %s is not closed: %v", path, err)
		}
		warningsByFilename[path] = warning
	}
	processDirectory := func(directory string, processFile func(string)) error {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if isFileFormatValid(path) {
				processFile(path)
			}

			return nil
		})
		return err
	}

	processedFiles := map[string]bool{}
	for _, filename := range files {
		var isDir bool
		if filename != "-" {
			fi, err := os.Stat(filename)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("cannot stat file %q: %v", filename, err))
				continue
			}
			isDir = fi.IsDir()
		}

		if !isDir {
			processFile(filename)
			processedFiles[filename] = true
			continue
		}
		if err := processDirectory(filename, func(path string) {
			processFile(path)
			processedFiles[path] = true
		}); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	files = []string{}
	for p := range processedFiles {
		files = append(files, p)
	}

	if errs != nil {
		for _, fname := range files {
			if w := warningsByFilename[fname]; w != nil {
				if fname == "-" {
					_, _ = fmt.Fprint(writer, warningToString(w))
					break
				}
				_, _ = fmt.Fprintf(writer, "%q has warnings: %v\n", fname, warningToString(w))
			}
		}
		return errs
	}
	for _, fname := range files {
		if fname == "-" {
			if w := warningsByFilename[fname]; w != nil {
				_, _ = fmt.Fprint(writer, warningToString(w))
			} else {
				_, _ = fmt.Fprintf(writer, "validation successfully completed\n")
			}
			break
		}

		if w := warningsByFilename[fname]; w != nil {
			_, _ = fmt.Fprintf(writer, "%q has warnings: %v\n", fname, warningToString(w))
		} else {
			_, _ = fmt.Fprintf(writer, "%q is valid\n", fname)
		}
	}

	return nil
}

func warningToString(w validation.Warning) string {
	we, ok := w.(*multierror.Error)
	if ok {
		we.ErrorFormat = func(i []error) string {
			points := make([]string, len(i))
			for i, err := range i {
				points[i] = fmt.Sprintf("* %s", err)
			}

			return fmt.Sprintf(
				"\n\t%s\n",
				strings.Join(points, "\n\t"))
		}
	}
	return w.Error()
}

func isFileFormatValid(file string) bool {
	ext := filepath.Ext(file)
	return slices.Contains(fileExtensions, ext)
}
