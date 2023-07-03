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

package controller

import (
	"context"

	"github.com/dubbogo/gost/encoding/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	trafficv1 "github.com/apache/dubbo-admin/pkg/traffic/api/v1"
	"github.com/apache/dubbo-admin/pkg/traffic/cache"
)

// DynamicConfigReconciler reconciles a DynamicConfig object
type DynamicConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=traffic.dubbo.apache.org,resources=dynamicconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=traffic.dubbo.apache.org,resources=dynamicconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=traffic.dubbo.apache.org,resources=dynamicconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DynamicConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *DynamicConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Enabled()
	dynamicConfig := &trafficv1.DynamicConfig{}
	err := r.Get(ctx, req.NamespacedName, dynamicConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			notify([]byte(""), constant.DynamicKey)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !dynamicConfig.DeletionTimestamp.IsZero() {
		notify([]byte(""), constant.DynamicKey)
		return ctrl.Result{}, nil
	}

	dynamic := dynamicConfig.Spec.DynamicConfig
	bytes, err := yaml.MarshalYML(dynamic)
	cache.ConfigMap[constant.DynamicKey] = string(bytes)
	if err != nil {
		return ctrl.Result{}, err
	}
	notify(bytes, constant.DynamicKey)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DynamicConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trafficv1.DynamicConfig{}).
		Complete(r)
}
