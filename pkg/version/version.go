package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	gitVersion   = "dubbo-admin-%s"
	gitCommit    = "$Format:%H$"
	gitTreeState = "" // state of git tree, either "clean" or "dirty"
	gitTag       = ""
	buildDate    = "1970-01-01T00:00:00Z"
)

type Version struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	GitTag       string `json:"gitTag"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

func GetVersion() Version {
	version := Version{
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		GitTag:       gitTag,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	return version
}

func GetVersionInfo() string {
	version := GetVersion()
	result, _ := json.Marshal(version)
	return string(result)
}
