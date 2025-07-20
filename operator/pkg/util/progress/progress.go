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

package progress

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/apache/dubbo-admin/operator/pkg/component"
	"github.com/cheggaaa/pb/v3"
)

type InstallState int

const inProgress = `{{ yellow (cycle . "-" "-" "-" " ") }} `

const (
	StateInstalling InstallState = iota
	StatePruning
	StateComplete
	StateUninstallComplete
)

// Log records the progress of an installation
// This aims to provide information about the install of multiple components in parallel, while working
// around the limitations of the pb library, which will only support single lines. To do this, we aggregate
// the current components into a single line, and as components complete there final state is persisted to a new line.
type Log struct {
	components map[string]*ManifestLog
	state      InstallState
	bar        *pb.ProgressBar
	mu         sync.Mutex
	template   string
}

func NewLog() *Log {
	return &Log{
		components: map[string]*ManifestLog{},
		bar:        createBar(),
	}
}

// createStatus will return a string to report the current status.
// ex: - Processing resources for components. Waiting for foo, bar.
func (i *Log) createStatus(maxWidth int) string {
	comps := make([]string, 0, len(i.components))
	wait := make([]string, 0, len(i.components))
	for c, l := range i.components {
		comps = append(comps, component.UserFacingCompName(component.Name(c)))
		wait = append(wait, l.waitingResources()...)
	}
	sort.Strings(comps)
	sort.Strings(wait)
	msg := fmt.Sprintf(`Processing resources for %s.`, strings.Join(comps, ", "))
	if len(wait) > 0 {
		msg += fmt.Sprintf(` Waiting for %s`, strings.Join(wait, ", "))
	}
	prefix := inProgress
	if !i.bar.GetBool(pb.Terminal) {
		// If we aren't a terminal, no need to spam extra lines
		prefix = `{{ yellow "-" }} `
	}
	// reduce by 2 to allow for the "- " that will be added below
	maxWidth -= 2
	if maxWidth > 0 && len(msg) > maxWidth {
		return prefix + msg[:maxWidth-3] + "..."
	}
	// cycle will alternate between "-" and " ". "-" is given multiple times to avoid quick flashing back and forth
	return prefix + msg
}

func (i *Log) NewComponent(comp string) *ManifestLog {
	mi := &ManifestLog{
		report: i.reportProgress(comp),
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.components[comp] = mi
	return mi
}

// reportProgress will report an update for a given component
// Because the bar library does not support multiple lines/bars at once, we need to aggregate current
// progress into a single line. For example "Waiting for x, y, z". Once a component completes, we want
// a new line created so the information is not lost. To do this, we spin up a new bar with the remaining components
// on a new line, and create a new bar. For example, this becomes "x succeeded", "waiting for y, z".
func (i *Log) reportProgress(componentName string) func() {
	return func() {
		compName := component.Name(componentName)
		cliName := component.UserFacingCompName(compName)
		i.mu.Lock()
		defer i.mu.Unlock()
		comp := i.components[componentName]
		// The component has completed
		comp.mu.Lock()
		finished := comp.finished
		compErr := comp.err
		comp.mu.Unlock()
		successIcon := "✅"
		if icon, found := component.Icons[compName]; found {
			successIcon = icon
		}
		if finished || compErr != "" {
			if finished {
				i.SetMessage(fmt.Sprintf(`{{ green "✔" }} %s install Completed %s`, cliName, successIcon), true)
			} else {
				i.SetMessage(fmt.Sprintf(`{{ read "✘" }} %s encountered an error: %s`, cliName, compErr), true)
			}
			// Close the bar out, outputting a new line
			delete(i.components, componentName)
			// Now we create a new bar, which will have the remaining components
			i.bar = createBar()
			return
		}
		i.SetMessage(i.createStatus(i.bar.Width()), false)
	}
}

func (i *Log) SetMessage(status string, finish bool) {
	// if we are not a terminal and there is no change, do not write
	// This avoids redundant lines
	if !i.bar.GetBool(pb.Terminal) && status == i.template {
		return
	}
	i.template = status
	i.bar.SetTemplateString(i.template)
	if finish {
		i.bar.Finish()
	}
	i.bar.Write()
}

func (i *Log) SetState(state InstallState) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.state = state
	switch i.state {
	case StatePruning:
		i.SetMessage(inProgress+`Pruning removed resources`, false)
	case StateComplete:
		// TODO Install provides the command for copying and running.
		i.SetMessage(`{{ green "✔" }} All Dubbo resources have been successfully installed to the cluster.`, true)
	case StateUninstallComplete:
		// TODO Uninstall provides a one-click command to delete CRDs.
		i.SetMessage(`{{ green "✔" }} All Dubbo resources have been successfully removed from the cluster.`, true)
	}
}

// For testing only
var testWriter *io.Writer

func createBar() *pb.ProgressBar {
	// Don't set a total and use Static so we can explicitly control when you write. This is needed
	// for handling the multiline issues.
	bar := pb.New(0)
	bar.Set(pb.Static, true)
	if testWriter != nil {
		bar.SetWriter(*testWriter)
	}
	bar.Start()
	// if we aren't a terminal, we will return a new line for each new message
	if !bar.GetBool(pb.Terminal) {
		bar.Set(pb.ReturnSymbol, "\n")
	}
	return bar
}

// ManifestLog records progress for a single component
type ManifestLog struct {
	report   func()
	err      string
	waiting  []string
	finished bool
	mu       sync.Mutex
}

func (mi *ManifestLog) ReportProgress() {
	if mi == nil {
		return
	}
}

func (mi *ManifestLog) ReportFinished() {
	if mi == nil {
		return
	}
	mi.mu.Lock()
	mi.finished = true
	mi.mu.Unlock()
	mi.report()
}

func (mi *ManifestLog) ReportError(err string) {
	if mi == nil {
		return
	}
	mi.mu.Lock()
	mi.err = err
	mi.mu.Unlock()
	mi.report()
}

func (mi *ManifestLog) ReportWaiting(resources []string) {
	if mi == nil {
		return
	}
	mi.mu.Lock()
	mi.waiting = resources
	mi.mu.Unlock()
	mi.report()
}

func (p *ManifestLog) waitingResources() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waiting
}
