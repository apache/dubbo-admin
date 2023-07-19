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

package subcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/app/dubboctl/internal/kube"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type ManifestDiffArgs struct {
	CompareDir bool
}

func ConfigManifestDiffCmd(baseCmd *cobra.Command) {
	mdArgs := &ManifestDiffArgs{}
	mdCmd := &cobra.Command{
		Use:   "diff",
		Short: "show the difference between two files or dirs",
		Example: `  # show the difference between two files
  dubboctl manifest diff fileA fileB
  # show the difference between two dirs
  dubboctl manifest diff dirA dirB --compareDir=true
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("manifest diff needs two files or dirs")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			if err := compare(args, mdArgs); err != nil {
				return err
			}
			return nil
		},
	}
	mdCmd.PersistentFlags().BoolVarP(&mdArgs.CompareDir, "compareDir", "", false,
		"Indicate whether compare two dirs or two files")

	baseCmd.AddCommand(mdCmd)
}

func compare(args []string, mdArgs *ManifestDiffArgs) error {
	var res string
	var err error
	if mdArgs.CompareDir {
		res, err = compareDirs(args[0], args[1])
	} else {
		res, err = compareFiles(args[0], args[1])
	}
	if err != nil {
		return err
	}
	logger.CmdSugar().Print(res)
	return nil
}

func compareDirs(dirA, dirB string) (string, error) {
	filesA, err := os.ReadDir(dirA)
	if err != nil {
		return "", err
	}
	filesB, err := os.ReadDir(dirB)
	if err != nil {
		return "", err
	}
	createFileMap := func(dir string, files []os.DirEntry) (map[string]struct{}, error) {
		res := make(map[string]struct{})
		for _, file := range files {
			if file.IsDir() {
				return nil, errors.New("do not support recursive traversal")
			}
			res[file.Name()] = struct{}{}
		}
		return res, nil
	}
	mapA, err := createFileMap(dirA, filesA)
	if err != nil {
		return "", err
	}
	mapB, err := createFileMap(dirB, filesB)
	if err != nil {
		return "", err
	}
	var diffBuilder strings.Builder
	var addBuilder strings.Builder
	var errBuilder strings.Builder
	for file := range mapA {
		if _, ok := mapB[file]; ok {
			fileA := filepath.Join(dirA, file)
			fileB := filepath.Join(dirB, file)
			res, err := compareFiles(fileA, fileB)
			if err != nil {
				errBuilder.WriteString(err.Error() + "\n")
				continue
			}
			if res != "" {
				diffBuilder.WriteString(fmt.Sprintf("%s---%s\n", fileA, fileB))
				diffBuilder.WriteString(res + "\n")
			}
		} else {
			addBuilder.WriteString(fmt.Sprintf("%s doesn't exist in %s\n", file, dirB))
		}
	}
	for file := range mapB {
		if _, ok := mapA[file]; !ok {
			addBuilder.WriteString(fmt.Sprintf("%s doesn't exist in %s\n", file, dirA))
		}
	}
	errRes := errBuilder.String()
	if errRes != "" {
		errRes = "------parse error------\n" + errRes + "\n"
	}
	addRes := addBuilder.String()
	if addRes != "" {
		addRes = "------addition------\n" + addRes + "\n"
	}
	diffRes := diffBuilder.String()
	if diffRes != "" {
		diffRes = "------diff------\n" + diffRes + "\n"
	}

	var final string
	if errRes == "" && addRes == "" && diffRes == "" {
		final = "two dirs are identical\n"
	} else {
		final = errRes + addRes + diffRes
	}

	return final, nil
}

func compareFiles(fileA, fileB string) (string, error) {
	bytesA, err := os.ReadFile(fileA)
	if err != nil {
		return "", fmt.Errorf("read %s failed, err: %s", fileA, err)
	}
	bytesB, err := os.ReadFile(fileB)
	if err != nil {
		return "", fmt.Errorf("read %s failed, err: %s", fileB, err)
	}
	objsA, err := kube.ParseObjectsFromManifest(string(bytesA), false)
	if err != nil {
		return "", fmt.Errorf("parse %s failed, err: %s", fileA, err)
	}
	objsB, err := kube.ParseObjectsFromManifest(string(bytesB), false)
	if err != nil {
		return "", fmt.Errorf("parse %s failed, err: %s", fileA, err)
	}
	diffRes, addRes, errRes := kube.CompareObjects(objsA, objsB)

	return diffRes + addRes + errRes, nil
}
