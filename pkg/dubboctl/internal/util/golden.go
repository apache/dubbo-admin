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
)

// TestYAMLEqual judges whether golden and result yaml are the same and return the diff if they are different.
// If this function returns error, it means that golden file or result file could not be formatted.
// eg:
//
//	 golden: line
//	 result: line
//	 return: true, ""
//
//	 golden: lineG
//	 result: lineR
//	 return: false, line 1 diff:
//	                --golden--
//	                lineG
//	                --result--
//	                lineR
//
//	golden: line
//	result: line
//	        lineAdd
//	return: false, line 2 to 2:
//	               --result addition--
//	               lineAdd
func TestYAMLEqual(golden, result string) (bool, string, error) {
	golden = ApplyFilters(golden, LicenseFilter, SpaceLineFilter)
	result = ApplyFilters(result, LicenseFilter, SpaceLineFilter)
	// do not use FormatterFilter here because we need to know whether manifest could be formatted
	var err error
	golden, err = formatTestYAML(golden)
	if err != nil {
		return false, "", fmt.Errorf("golden file format err: %s", err)
	}
	result, err = formatTestYAML(result)
	if err != nil {
		return false, "", fmt.Errorf("result file format err: %s", err)
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
		// judge whether lineG and lindR are the same. if not, add this diff line to diffBuilder.
		if !isTestYAMLLineEqual(lineG, lineR) {
			diffFlag = true
			diffBuilder.WriteString(
				fmt.Sprintf("line %d diff:\n--golden--\n%s\n--result--\n%s\n", line, lineG, lineR),
			)
		}
	}
	// additional lines processing
	// [lineStart, lineEnd] represents additional lines

	// scan golden ends.
	if len(scannerG.Text()) == 0 {
		var addBuilder strings.Builder
		lineStart, lineEnd := line+1, line
		for scannerR.Scan() {
			lineEnd += 1
			addBuilder.WriteString(scannerR.Text() + "\n")
		}
		// length of result is equal to length of golden
		if lineStart > lineEnd {
			// result is equal to golden
			if !diffFlag {
				return true, "", nil
			}
			return false, diffBuilder.String(), nil
		}
		// result is longer than golden, we add these additional lines to diffBuilder
		diffBuilder.WriteString(
			fmt.Sprintf("line %d to %d:\n--result addition--\n%s", lineStart, lineEnd, addBuilder.String()),
		)
		return false, diffBuilder.String(), nil
	}

	// scan result ends, we know that golden is longer than result.
	// due to scannerG.Scan has been invoked before, we add the first additional line.
	var addBuilder strings.Builder
	lineStart, lineEnd := line+1, line+1
	addBuilder.WriteString(scannerG.Text() + "\n")
	for scannerG.Scan() {
		lineEnd += 1
		addBuilder.WriteString(scannerG.Text() + "\n")
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
	keyG, valG, typG := splitKeyVal(golden)
	keyR, valR, typR := splitKeyVal(result)
	if typG != typR {
		return false
	}
	if keyG != keyR {
		return false
	}
	// golden and result strings could not be split by ":", compare them directly
	if typG == complete {
		return true
	}
	if valG == valR {
		return true
	}
	// valG may contain regular expression
	reg, err := regexp.Compile(valG)
	if err != nil {
		return false
	}
	if !reg.MatchString(result) {
		return false
	}
	return true
}

type keyValType uint8

const (
	// eg: line
	complete keyValType = iota + 1
	// eg: key:val
	singlePair
	// eg: key:val:val
	multiPairs
)

// split key:val format str and return keyValType.
// eg:
//
//		 key,          it returns key, "", complete.
//	     key:val,      it returns key, val, singlePair
//		 key:val:val1, it returns key, val:val1, multiPairs;
func splitKeyVal(str string) (string, string, keyValType) {
	var key, val string
	var typ keyValType
	elems := strings.Split(str, ":")
	switch len(elems) {
	case 1:
		key = elems[0]
		typ = complete
	case 2:
		key = elems[0]
		val = elems[1]
		typ = singlePair
	default:
		key = elems[0]
		val = strings.Join(elems[1:], "")
		typ = multiPairs
	}
	return strings.TrimSpace(key), strings.TrimSpace(val), typ
}
