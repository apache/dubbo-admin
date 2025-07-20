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

package deployer

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"os"
	template2 "text/template"
)

const (
	deployTemplateFile = "deploy.tpl"
)

//go:embed deploy.tpl
var deployTemplate string

type deploy struct{}

type Deployment struct {
	Name       string
	Namespace  string
	Image      string
	Port       int
	TargetPort int
	NodePort   int
}

type DeployerOption func(deployer *deploy)

func NewDeployer(opts ...DeployerOption) *deploy {
	d := &deploy{}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *deploy) Deploy(ctx context.Context, dc *dubbo.DubboConfig, option ...sdk.DeployOption) (sdk.DeploymentResult, error) {
	ns := dc.Deploy.Namespace

	var err error
	text, err := util.LoadTemplate("", deployTemplateFile, deployTemplate)
	if err != nil {
		return sdk.DeploymentResult{
			Status:    sdk.Failed,
			Namespace: ns,
		}, err
	}

	targetPort := dc.Deploy.TargetPort
	if targetPort == 0 {
		targetPort = dc.Deploy.Port
	}

	path := dc.Root + "/" + dc.Deploy.Output
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		fmt.Println("The k8s yaml file already exists in this directory.")
	}

	out, err := os.Create(path)
	if err != nil {
		return sdk.DeploymentResult{
			Status:    sdk.Failed,
			Namespace: ns,
		}, err
	}

	t := template2.Must(template2.New("deployTemplate").Parse(text))
	err = t.Execute(out, Deployment{
		Name:       dc.Name,
		Namespace:  ns,
		Image:      dc.Image,
		Port:       dc.Deploy.Port,
		TargetPort: targetPort,
		NodePort:   dc.Deploy.NodePort,
	})
	if err != nil {
		return sdk.DeploymentResult{
			Status:    sdk.Failed,
			Namespace: ns,
		}, err
	}

	return sdk.DeploymentResult{
		Status:    sdk.Deployed,
		Namespace: ns,
	}, nil
}
