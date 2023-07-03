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

package cmd

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	//+kubebuilder:scaffold:imports

	trafficv1 "github.com/apache/dubbo-admin/pkg/traffic/api/v1"
	"github.com/apache/dubbo-admin/pkg/traffic/cache"
	"github.com/apache/dubbo-admin/pkg/traffic/internal/watchserver"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	cache.Init()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(trafficv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func Start() {

	gc := watchserver.RegisterGrpc()
	defer func() {
		gc.Stop()
	}()

	StartController()
}
