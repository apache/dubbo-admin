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

package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	buildVersion = "unknown"
)

// type BuildInfo struct {
//	Version string `json:"version"`
// }

func (b BuildInfo) String() string {
	return fmt.Sprintf("%v", b.Version)
}

func (b BuildInfo) LongForm() string {
	return fmt.Sprintf("%#v", b)
}

func init() {
	Info = BuildInfo{
		Version: buildVersion,
	}
}

var (
	Product      = "Dubbo"
	basedOndubbo = ""
	version      = "unknown"
	gitTag       = "unknown"
	gitCommit    = "unknown"
	buildDate    = "unknown"
	Envoy        = "unknown"
)

type BuildInfo struct {
	Product      string
	Version      string
	GitTag       string
	GitCommit    string
	BuildDate    string
	BasedOnDubbo string
}

func (b BuildInfo) FormatDetailedProductInfo() string {
	base := []string{
		fmt.Sprintf("Product:       %s", b.Product),
		fmt.Sprintf("Version:       %s", b.Version),
		fmt.Sprintf("Git Tag:       %s", b.GitTag),
		fmt.Sprintf("Git Commit:    %s", b.GitCommit),
		fmt.Sprintf("Build Date:    %s", b.BuildDate),
	}
	if b.BasedOnDubbo != "" {
		base = append(base, fmt.Sprintf("Based on dubbo: %s", b.BasedOnDubbo))
	}
	return strings.Join(
		base,
		"\n",
	)
}

func shortCommit(c string) string {
	if len(c) < 7 {
		return c
	}
	return c[:7]
}

func (b BuildInfo) AsMap() map[string]string {
	res := map[string]string{
		"product":    b.Product,
		"version":    b.Version,
		"build_date": b.BuildDate,
		"git_commit": shortCommit(b.GitCommit),
		"git_tag":    b.GitTag,
	}
	if b.BasedOnDubbo != "" {
		res["based_on_dubbo"] = b.BasedOnDubbo
	}
	return res
}

func (b BuildInfo) UserAgent(component string) string {
	commit := shortCommit(b.GitCommit)
	if b.BasedOnDubbo != "" {
		commit = fmt.Sprintf("%s/dubbo-%s", commit, b.BasedOnDubbo)
	}
	return fmt.Sprintf("%s/%s (%s; %s; %s/%s)",
		component,
		b.Version,
		runtime.GOOS,
		runtime.GOARCH,
		b.Product,
		commit)
}

var Build BuildInfo

func init() {
	Build = BuildInfo{
		Product:      Product,
		Version:      version,
		GitTag:       gitTag,
		GitCommit:    gitCommit,
		BuildDate:    buildDate,
		BasedOnDubbo: basedOndubbo,
	}
}
