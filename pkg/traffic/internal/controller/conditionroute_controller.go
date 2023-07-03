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

// ConditionRouteReconciler reconciles a ConditionRoute object
type ConditionRouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=traffic.dubbo.apache.org,resources=conditionroutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=traffic.dubbo.apache.org,resources=conditionroutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=traffic.dubbo.apache.org,resources=conditionroutes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConditionRoute object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ConditionRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Enabled()
	conditionRoute := &trafficv1.ConditionRoute{}
	err := r.Get(ctx, req.NamespacedName, conditionRoute)
	if err != nil {
		if errors.IsNotFound(err) {
			notify([]byte(""), constant.ConditionRoute)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !conditionRoute.DeletionTimestamp.IsZero() {
		notify([]byte(""), constant.ConditionRoute)
		return ctrl.Result{}, nil
	}

	condition := conditionRoute.Spec.ConditionRoute
	bytes, err := yaml.MarshalYML(condition)
	cache.ConfigMap[constant.ConditionRoute] = string(bytes)
	if err != nil {
		return ctrl.Result{}, err
	}
	notify(bytes, constant.ConditionRoute)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConditionRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trafficv1.ConditionRoute{}).
		Complete(r)
}
