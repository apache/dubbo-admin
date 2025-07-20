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
	"context"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"strings"
)

const DefaultRepositoryName = "common"

type Templates struct {
	client *Client
}

func newTemplates(client *Client) *Templates {
	return &Templates{
		client: client,
	}
}

func (t *Templates) List(runtime string) ([]string, error) {
	var names []string
	extended := util.NewSortedSet()

	rr, err := t.client.Repositories().All()
	if err != nil {
		return []string{}, err
	}

	for _, r := range rr {
		tt, err := r.Templates(runtime)
		if err != nil {
			return []string{}, err
		}
		for _, t := range tt {
			if r.Name == DefaultRepositoryName {
				names = append(names, t.Name())
			} else {
				extended.Add(t.Fullname())
			}
		}
	}
	return append(names, extended.Items()...), nil
}

func (t *Templates) Get(runtime, fullname string) (Template, error) {
	var (
		template Template
		repo     Repository
		err      error
	)

	repoName, tplName := splitTemplateFullname(fullname)

	repo, err = t.client.repositories.Get(repoName)
	if err != nil {
		return template, err
	}

	return repo.Template(runtime, tplName)
}

func (t *Templates) Write(dc *dubbo.DubboConfig) error {
	if err := dc.Validate(); err != nil {
		return err
	}
	template, err := t.Get(dc.Runtime, dc.Template)
	if err != nil {
		return err
	}
	return template.Write(context.TODO(), dc)
}

func splitTemplateFullname(name string) (repoName, tplName string) {
	cc := strings.Split(name, "/")
	if len(cc) == 1 {
		repoName = DefaultRepositoryName
		tplName = name
	} else {
		repoName = cc[0]
		tplName = cc[1]
	}
	return
}
