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

package context

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"
)

type Variable struct {
	Required bool
	MaxBytes int
}
type Template struct {
	AllowedTools                     []string
	ID, Version, Shared, Act, Answer string
	Variables                        map[string]Variable
}

// Ref is the immutable content identity used to pin a Session. The runtime tool
// allowlist is resolved from the current registry at compile time and is
// deliberately excluded: tool discovery or registry changes must not invalidate
// Sessions pinned to an unchanged prompt version.
func (t Template) Ref() string {
	identity := t
	identity.AllowedTools = nil
	return fmt.Sprintf("%s@%s:%x", t.ID, t.Version, sha256.Sum256(JSON(identity)))
}

func (t Template) Render(phase string, values map[string]string) (string, error) {
	if len(t.Variables) > 64 {
		return "", fmt.Errorf("too many template variables")
	}
	inputs := map[string]string{}
	for key, value := range values {
		rule, ok := t.Variables[key]
		if !ok || rule.MaxBytes <= 0 || len(value) > rule.MaxBytes {
			return "", fmt.Errorf("invalid template variable %q", key)
		}
		inputs[key] = value
	}
	for key, rule := range t.Variables {
		if rule.MaxBytes <= 0 {
			return "", fmt.Errorf("invalid template variable limit")
		}
		if value, ok := values[key]; rule.Required && (!ok || value == "") {
			return "", fmt.Errorf("%w: %s", ErrTemplateInput, key)
		}
		if _, ok := inputs[key]; !ok {
			inputs[key] = ""
		}
	}
	policy := t.Act
	if phase == "answer" {
		policy = t.Answer
	} else if phase != "act" {
		return "", fmt.Errorf("invalid phase")
	}
	source := strings.TrimSpace(t.Shared) + "\n\n" + strings.TrimSpace(policy)
	tmpl, err := template.New(t.ID).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", err
	}
	if len(tmpl.Templates()) != 1 {
		return "", fmt.Errorf("template definitions are not permitted")
	}
	// Only literal text and {{.DeclaredVariable}} are allowed. In particular,
	// methods, functions, loops, pipelines, template inclusion and indexing fail.
	for _, node := range tmpl.Tree.Root.Nodes {
		switch n := node.(type) {
		case *parse.TextNode:
		case *parse.ActionNode:
			if len(n.Pipe.Decl) != 0 || len(n.Pipe.Cmds) != 1 || len(n.Pipe.Cmds[0].Args) != 1 {
				return "", fmt.Errorf("unsupported template action")
			}
			field, ok := n.Pipe.Cmds[0].Args[0].(*parse.FieldNode)
			if !ok || len(field.Ident) != 1 {
				return "", fmt.Errorf("unsupported template field")
			}
			if _, ok := t.Variables[field.Ident[0]]; !ok {
				return "", fmt.Errorf("undeclared template variable")
			}
		default:
			return "", fmt.Errorf("unsupported template node")
		}
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, inputs); err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateInput, err)
	}
	if buf.Len() > 128<<10 {
		return "", fmt.Errorf("rendered template too large")
	}
	return buf.String(), nil
}
