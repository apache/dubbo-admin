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

package util

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/yamlfmt/formatters/basic"
)

var (
	formatterConfig = func() *basic.Config {
		cfg := basic.DefaultConfig()
		return cfg
	}()
	formatter = &basic.BasicFormatter{
		Config:   formatterConfig,
		Features: basic.ConfigureFeaturesFromConfig(formatterConfig),
	}
)

func TestYAMLEqual(golden, result string) (bool, string, error) {
	var err error
	golden, err = formatTestYAML(golden)
	if err != nil {
		return false, "", err
	}
	result, err = formatTestYAML(result)
	if err != nil {
		return false, "", err
	}

	var diffBuilder strings.Builder
	var diffFlag bool
	var line int
	scannerG := bufio.NewScanner(strings.NewReader(strings.TrimSpace(golden)))
	scannerR := bufio.NewScanner(strings.NewReader(strings.TrimSpace(result)))
	for scannerG.Scan() && scannerR.Scan() {
		line += 1
		lineG := scannerG.Text()
		lineR := scannerR.Text()
		if !isTestYAMLLineEqual(lineG, lineR) {
			diffFlag = true
			diffBuilder.WriteString(
				fmt.Sprintf("line %d diff:\n--golden--\n%s\n--result--\n%s\n", line, lineG, lineR),
			)
		}
	}
	if len(scannerG.Text()) == 0 {
		var addBuilder strings.Builder
		lineStart, lineEnd := line+1, line
		for scannerR.Scan() {
			lineEnd += 1
			addBuilder.WriteString(scannerR.Text() + "\n")
		}
		if lineStart > lineEnd {
			if !diffFlag {
				return true, "", nil
			}
			return false, diffBuilder.String(), nil
		}
		diffBuilder.WriteString(
			fmt.Sprintf("line %d to %d:\n--result addition--\n%s", lineStart, lineEnd, addBuilder.String()),
		)
		return false, diffBuilder.String(), nil
	}

	var addBuilder strings.Builder
	lineStart, lineEnd := line+1, line+1
	addBuilder.WriteString(scannerG.Text() + "\n")
	for scannerG.Scan() {
		lineEnd += 1
		addBuilder.WriteString(scannerR.Text() + "\n")
	}
	diffBuilder.WriteString(
		fmt.Sprintf("line %d to %d:\n--golden addition--\n%s", lineStart, lineEnd, addBuilder.String()),
	)
	return false, diffBuilder.String(), nil
}

func formatTestYAML(original string) (string, error) {
	resBytes, err := formatter.Format([]byte(original))
	if err != nil {
		return "", err
	}
	return string(resBytes), nil
}

func isTestYAMLLineEqual(golden, result string) bool {
	keyG, valG, flagG := splitKeyVal(golden)
	keyR, valR, flagR := splitKeyVal(result)
	if flagG != flagR {
		return false
	}
	if keyG != keyR {
		return false
	}
	// golden and result strings could not be split by ":", compare them directly
	if flagG == false {
		return true
	}
	if valG == valR {
		return true
	}
	reg, err := regexp.Compile(valG)
	if err != nil {
		return false
	}
	if !reg.MatchString(result) {
		return false
	}
	return true
}

// split key:val format str and return whether this str could be split by ":".
// if str is not with this format, eg:
//
//	key:val:val1, it returns key, val:val1, true;
//	key,          it returns key, "", false.
//
// do not trim space cause that's a part of key and val.
func splitKeyVal(str string) (string, string, bool) {
	var key, val string
	var flag bool
	elems := strings.Split(str, ":")
	switch len(elems) {
	case 1:
		key = elems[0]
	case 2:
		key = elems[0]
		val = elems[1]
		flag = true
	default:
		key = elems[0]
		val = strings.Join(elems[1:], "")
		flag = true
	}
	return strings.TrimSpace(key), strings.TrimSpace(val), flag
}
