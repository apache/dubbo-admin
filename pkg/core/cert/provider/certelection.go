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

package provider

import (
	"context"
	"time"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type LeaderElection interface {
	Election(storage *CertStorage, options *dubbo_cp.Config, kubeClient kubernetes.Interface) error
}

type leaderElectionImpl struct{}

func NewleaderElection() LeaderElection {
	return &leaderElectionImpl{}
}

func (c *leaderElectionImpl) Election(storage *CertStorage, options *dubbo_cp.Config, kubeClient kubernetes.Interface) error {
	identity := options.Security.ResourcelockIdentity
	rlConfig := resourcelock.ResourceLockConfig{
		Identity: identity,
	}
	namespace := options.KubeConfig.Namespace
	_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		namespace = "default"
	}
	lock, err := resourcelock.New(resourcelock.ConfigMapsLeasesResourceLock, namespace, "dubbo-lock-cert", kubeClient.CoreV1(), kubeClient.CoordinationV1(), rlConfig)
	if err != nil {
		return err
	}
	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			// leader
			OnStartedLeading: func(ctx context.Context) {
				// lock if multi cp-server，refresh signed cert
				storage.SetAuthorityCert(GenerateAuthorityCert(storage.GetRootCert(), options.Security.CaValidity))
			},
			// not leader
			OnStoppedLeading: func() {
				// TODO should be listen,when cert resfresh,should be resfresh
			},
			// a new leader has been elected
			OnNewLeader: func(identity string) {
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	leaderelection.RunOrDie(ctx, leaderElectionConfig)
	return nil
}
