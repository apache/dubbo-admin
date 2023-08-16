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

package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/api/mesh"
	cert "github.com/apache/dubbo-admin/pkg/core/cert/provider"
	endpoint2 "github.com/apache/dubbo-admin/pkg/core/tools/endpoint"
	"google.golang.org/grpc/peer"

	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	apisv1alpha1 "github.com/apache/dubbo-admin/pkg/core/gen/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/gen/generated/clientset/versioned"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/snp/model"
	"github.com/pkg/errors"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RegisterRequest struct {
	ConfigsUpdated map[model.ConfigKey]map[string]struct{}
}

type Snp struct {
	mesh.UnimplementedServiceNameMappingServiceServer

	queue       chan *RegisterRequest
	config      *dubbo_cp.Config
	CertClient  cert.Client
	CertStorage *cert.CertStorage

	KubeClient versioned.Interface
}

func (s *Snp) Start(stop <-chan struct{}) error {
	go s.debounce(stop, s.push)
	return nil
}

func (s *Snp) NeedLeaderElection() bool {
	return false
}

func (s *Snp) RegisterServiceAppMapping(ctx context.Context, req *mesh.ServiceMappingRequest) (*mesh.ServiceMappingResponse, error) {
	namespace := req.GetNamespace()
	interfaces := req.GetInterfaceNames()
	applicationName := req.GetApplicationName()

	p, _ := peer.FromContext(ctx)
	_, err := endpoint2.ExactEndpoint(ctx, s.CertStorage, s.config, s.CertClient)
	if err != nil {
		logger.Sugar().Warnf("[ServiceMapping] Failed to exact endpoint from context: %v. RemoteAddr: %s", err, p.Addr.String())
		return &mesh.ServiceMappingResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	registerReq := &RegisterRequest{ConfigsUpdated: map[model.ConfigKey]map[string]struct{}{}}
	for _, interfaceName := range interfaces {
		key := model.ConfigKey{
			Name:      interfaceName,
			Namespace: namespace,
		}
		if _, ok := registerReq.ConfigsUpdated[key]; !ok {
			registerReq.ConfigsUpdated[key] = make(map[string]struct{})
		}
		registerReq.ConfigsUpdated[key][applicationName] = struct{}{}
	}
	s.queue <- registerReq

	return &mesh.ServiceMappingResponse{
		Success: true,
		Message: "success",
	}, nil
}

func NewSnp(config *dubbo_cp.Config, kubeClient versioned.Interface) *Snp {
	return &Snp{
		queue:      make(chan *RegisterRequest, 10),
		config:     config,
		KubeClient: kubeClient,
	}
}

func (r *RegisterRequest) Merge(req *RegisterRequest) *RegisterRequest {
	if r == nil {
		return req
	}
	for key, newApps := range req.ConfigsUpdated {
		if _, ok := r.ConfigsUpdated[key]; !ok {
			r.ConfigsUpdated[key] = make(map[string]struct{})
		}
		for app := range newApps {
			r.ConfigsUpdated[key][app] = struct{}{}
		}
	}
	return r
}

func (s *Snp) push(req *RegisterRequest) {
	for key, m := range req.ConfigsUpdated {
		var appNames []string
		for app := range m {
			appNames = append(appNames, app)
		}
		for i := 0; i < 3; i++ {
			if err := tryRegister(s.KubeClient, key.Namespace, key.Name, appNames); err != nil {
				logger.Errorf("[ServiceMapping] register [%v] failed: %v, try again later", key, err)
			} else {
				break
			}
		}
	}
}

func (s *Snp) debounce(stopCh <-chan struct{}, pushFn func(req *RegisterRequest)) {
	ch := s.queue
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastConfigUpdateTime time.Time

	pushCounter := 0
	debouncedEvents := 0

	var req *RegisterRequest

	free := true
	freeCh := make(chan struct{}, 1)

	push := func(req *RegisterRequest) {
		pushFn(req)
		freeCh <- struct{}{}
	}

	pushWorker := func() {
		eventDelay := time.Since(startDebounce)
		quietTime := time.Since(lastConfigUpdateTime)
		if eventDelay >= s.config.Options.DebounceMax || quietTime >= s.config.Options.DebounceAfter {
			if req != nil {
				pushCounter++

				if req.ConfigsUpdated != nil {
					logger.Infof("[ServiceMapping] Push debounce stable[%d] %d for config %s: %v since last change, %v since last push",
						pushCounter, debouncedEvents, configsUpdated(req),
						quietTime, eventDelay)
				}
				free = false
				go push(req)
				req = nil
				debouncedEvents = 0
			}
		} else {
			timeChan = time.After(s.config.Options.DebounceAfter - quietTime)
		}
	}

	for {
		select {
		case <-freeCh:
			free = true
			pushWorker()
		case r := <-ch:
			if !s.config.Options.EnableDebounce {
				go push(r)
				req = nil
				continue
			}

			lastConfigUpdateTime = time.Now()
			if debouncedEvents == 0 {
				timeChan = time.After(200 * time.Millisecond)
				startDebounce = lastConfigUpdateTime
			}
			debouncedEvents++

			req = req.Merge(r)
		case <-timeChan:
			if free {
				pushWorker()
			}
		case <-stopCh:
			return
		}
	}
}

func getOrCreateSnp(kubeClient versioned.Interface, namespace string, interfaceName string, newApps []string) (*apisv1alpha1.ServiceNameMapping, bool, error) {
	ctx := context.TODO()
	lowerCaseName := strings.ToLower(strings.ReplaceAll(interfaceName, ".", "-"))
	snpInterface := kubeClient.DubboV1alpha1().ServiceNameMappings(namespace)
	snp, err := snpInterface.Get(ctx, lowerCaseName, v1.GetOptions{})
	if err != nil {
		if apierror.IsNotFound(err) {
			snp, err = snpInterface.Create(ctx, &apisv1alpha1.ServiceNameMapping{
				ObjectMeta: v1.ObjectMeta{
					Name:      lowerCaseName,
					Namespace: namespace,
					Labels: map[string]string{
						"interface": interfaceName,
					},
				},
				Spec: api.ServiceNameMapping{
					InterfaceName:    interfaceName,
					ApplicationNames: newApps,
				},
			}, v1.CreateOptions{})
			if err == nil {
				logger.Debugf("create snp %s revision %s", interfaceName, snp.ResourceVersion)
				return snp, true, nil
			}
			if apierror.IsAlreadyExists(err) {
				logger.Debugf("[%s] has been exists, err: %v", err)
				snp, err = snpInterface.Get(ctx, lowerCaseName, v1.GetOptions{})
				if err != nil {
					return nil, false, errors.Wrap(err, "tryRegister retry get snp error")
				}
			}
		} else {
			return nil, false, errors.Wrap(err, "tryRegister get snp error")
		}
	}
	return snp, false, nil
}

func tryRegister(kubeClient versioned.Interface, namespace, interfaceName string, newApps []string) error {
	logger.Debugf("[ServiceMapping] try register [%s] in namespace [%s] with [%v] apps", interfaceName, namespace, len(newApps))
	snp, created, err := getOrCreateSnp(kubeClient, namespace, interfaceName, newApps)
	if created {
		logger.Debugf("[ServiceMapping] register success, revision:%s", snp.ResourceVersion)
		return nil
	}
	if err != nil {
		return err
	}

	previousLen := len(snp.Spec.ApplicationNames)
	previousAppNames := make(map[string]struct{}, previousLen)
	for _, name := range snp.Spec.ApplicationNames {
		previousAppNames[name] = struct{}{}
	}
	for _, newApp := range newApps {
		previousAppNames[newApp] = struct{}{}
	}
	if len(previousAppNames) == previousLen {
		logger.Debugf("[ServiceMapping] [%s] has been registered: %v", interfaceName, newApps)
		return nil
	}

	mergedApps := make([]string, 0, len(previousAppNames))
	for name := range previousAppNames {
		mergedApps = append(mergedApps, name)
	}
	snp.Spec.ApplicationNames = mergedApps
	snpInterface := kubeClient.DubboV1alpha1().ServiceNameMappings(namespace)
	snp, err = snpInterface.Update(context.Background(), snp, v1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, " update failed")
	}
	logger.Debugf("[ServiceMapping] register update success, revision:%s", snp.ResourceVersion)
	return nil
}

func configsUpdated(req *RegisterRequest) string {
	configs := ""
	for key := range req.ConfigsUpdated {
		configs += key.Name + key.Namespace
		break
	}
	if len(req.ConfigsUpdated) > 1 {
		more := fmt.Sprintf(" and %d more configs", len(req.ConfigsUpdated)-1)
		configs += more
	}
	return configs
}
