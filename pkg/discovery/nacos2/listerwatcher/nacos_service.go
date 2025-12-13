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

package listerwatcher

import (
	"fmt"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-co-op/gocron"
	nacosnamingclient "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	nacosmodel "github.com/nacos-group/nacos-sdk-go/v2/model"
	nacosvo "github.com/nacos-group/nacos-sdk-go/v2/vo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type NacosServiceListerWatcher struct {
	cfg          *discoverycfg.Config
	namingClient nacosnamingclient.INamingClient
	// watchingServices is used to record the services that are currently being watched
	// key -> nacos service name, value -> true if it is watched in current list turn
	watchingServices map[string]bool
	scheduler        *gocron.Scheduler
	resultChan       chan watch.Event
	stopWatch        bool
}

func NewNacosServiceListerWatcher(cfg *discoverycfg.Config, namingClient nacosnamingclient.INamingClient) *NacosServiceListerWatcher {
	return &NacosServiceListerWatcher{
		cfg:              cfg,
		namingClient:     namingClient,
		resultChan:       make(chan watch.Event),
		watchingServices: make(map[string]bool),
		scheduler:        gocron.NewScheduler(time.UTC),
		stopWatch:        false,
	}
}

func (lw *NacosServiceListerWatcher) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	serviceNames, err := lw.fetchAllServiceNames()
	if err != nil {
		return nil, err
	}
	nacosServiceList := meshresource.NewNacosServiceResourceListWithItems()
	resList := make([]*meshresource.NacosServiceResource, 0)
	for _, serviceName := range serviceNames {
		service, err := lw.namingClient.GetService(nacosvo.GetServiceParam{
			ServiceName: serviceName,
		})
		if err != nil {
			logger.Errorf("get instances of service %s failed in nacos %s, cause: %v", serviceName, lw.nacosAddress(), err)
			continue
		}
		resList = append(resList, lw.toNacosServiceResource(serviceName, service.Hosts))
	}
	nacosServiceList.Items = resList
	return nacosServiceList, nil
}

func (lw *NacosServiceListerWatcher) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	_, err := lw.scheduler.Every(lw.cfg.Properties.ServiceWatchPeriod).Seconds().Do(func() {
		if lw.stopWatch {
			logger.Debugf("stop watch all services of nacos %s", lw.mesh())
			lw.scheduler.Stop()
			return
		}
		startTime := time.Now()
		err := lw.fetchAndProcessService()
		if err != nil {
			logger.Errorf("fetch all service failed in nacos %s, cause: %v", lw.nacosAddress(), err)
			return
		}
		costs := time.Now().UnixMilli() - startTime.UnixMilli()
		logger.Debugf("finish fetching and processing all services in nacos %s, costs %dms", lw.nacosAddress(), costs)
	})
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.UnknownError,
			fmt.Sprintf("watch all services of nacos %s in a schedule occurs error", lw.mesh()))
	}
	lw.scheduler.StartAsync()
	return lw, nil
}

func (lw *NacosServiceListerWatcher) fetchAllServiceNames() ([]string, error) {
	serviceNameList := make([]string, 0)
	var pageNum uint32 = 1
	var pageSize uint32 = 100
	// If the number of serviceInfos is less than the page size, it means that the last page has been reached
	for {
		serviceNames, err := lw.listPage(pageNum, pageSize)
		if err != nil {
			return nil, err
		}
		serviceNameList = append(serviceNameList, serviceNames...)
		if len(serviceNames) < int(pageSize) {
			break
		}
		pageNum++
	}
	return serviceNameList, nil
}

func (lw *NacosServiceListerWatcher) fetchAndProcessService() error {
	var pageNum uint32 = 1
	var pageSize uint32 = 100
	// list all services by page search and subscribe it
	for {
		serviceNames, err := lw.listPage(pageNum, pageSize)
		if err != nil {
			return err
		}
		slice.ForEach(serviceNames, func(index int, item string) {
			lw.processNacosService(item)
		})
		if len(serviceNames) < int(pageSize) {
			break
		}
		pageNum++
	}
	// remove subscribers for services that are no longer being watched
	for serviceName, watched := range lw.watchingServices {
		if watched {
			lw.watchingServices[serviceName] = false
			continue
		}
		err := lw.unsubscribeService(serviceName)
		if err != nil {
			logger.Errorf("unsubscribe service %s failed in nacos %s, cause: %v", serviceName, lw.nacosAddress(), err)
			continue
		}
		// delete service from watchingServices, only one go routine, thread safe
		delete(lw.watchingServices, serviceName)
	}
	return nil
}

func (lw *NacosServiceListerWatcher) processNacosService(serviceName string) {
	if _, exists := lw.watchingServices[serviceName]; exists {
		lw.watchingServices[serviceName] = true
		return
	}
	err := lw.subscribeService(serviceName)
	if err != nil {
		logger.Errorf("subscribe service %s failed in nacos %s, cause: %v", serviceName, lw.nacosAddress(), err)
		return
	}
	lw.watchingServices[serviceName] = true
}

func (lw *NacosServiceListerWatcher) listPage(pageNo, pageSize uint32) ([]string, error) {
	serviceNames := make([]string, 0, pageSize)
	serviceInfos, err := lw.namingClient.GetAllServicesInfo(nacosvo.GetAllServiceInfoParam{
		PageSize: pageSize,
		PageNo:   pageNo,
	})

	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("get service names failed in nacos %s failed, page size: %d, page num: %d", lw.nacosAddress(), pageSize, pageNo))
	}
	serviceNames = append(serviceNames, serviceInfos.Doms...)

	return serviceNames, nil
}

func (lw *NacosServiceListerWatcher) subscribeService(serviceName string) error {
	logger.Infof("subscribe service %s in nacos %s", serviceName, lw.mesh())
	err := lw.namingClient.Subscribe(&nacosvo.SubscribeParam{
		ServiceName: serviceName,
		SubscribeCallback: func(instances []nacosmodel.Instance, err error) {
			if err != nil {
				logger.Errorf("subscribe service %s failed in nacos %s, cause: %v", serviceName, lw.nacosAddress(), err)
				return
			}
			lw.resultChan <- watch.Event{
				Type:   watch.Modified,
				Object: lw.toNacosServiceResource(serviceName, instances),
			}
		},
	})
	if err != nil {
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("subscribe service %s failed in nacos %s", serviceName, lw.mesh()))
	}
	return nil
}

func (lw *NacosServiceListerWatcher) unsubscribeService(serviceName string) error {
	logger.Infof("unsubscribe service %s in nacos %s", serviceName, lw.mesh())
	lw.resultChan <- watch.Event{
		Type:   watch.Deleted,
		Object: lw.toNacosServiceResource(serviceName, []nacosmodel.Instance{}),
	}
	return lw.namingClient.Unsubscribe(&nacosvo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   "DEFAULT_GROUP",
		SubscribeCallback: func(instances []nacosmodel.Instance, err error) {
			if err != nil {
				logger.Errorf("unsubscribe service %s failed in nacos %s, cause: %v", serviceName, lw.nacosAddress(), err)
				return
			}
			lw.resultChan <- watch.Event{
				Type:   watch.Deleted,
				Object: lw.toNacosServiceResource(serviceName, instances),
			}
		},
	})
}

func (lw *NacosServiceListerWatcher) toNacosServiceResource(serviceName string, instances []nacosmodel.Instance) *meshresource.NacosServiceResource {
	resource := meshresource.NewNacosServiceResourceWithAttributes(serviceName, lw.cfg.Name)
	nacosInstances := slice.Map(instances, func(index int, item nacosmodel.Instance) *meshproto.NacosInstance {
		return &meshproto.NacosInstance{
			Ip:       item.Ip,
			Port:     int64(item.Port),
			Metadata: item.Metadata,
		}
	})
	resource.Spec = &meshproto.NacosService{
		ServiceKey: serviceName,
		Instances:  nacosInstances,
	}
	return resource
}

func (lw *NacosServiceListerWatcher) ResourceKind() coremodel.ResourceKind {
	return meshresource.NacosServiceKind
}

func (lw *NacosServiceListerWatcher) TransformFunc() cache.TransformFunc {
	return nil
}

func (lw *NacosServiceListerWatcher) Stop() {
	lw.stopWatch = true
}

func (lw *NacosServiceListerWatcher) ResultChan() <-chan watch.Event {
	return lw.resultChan
}

func (lw *NacosServiceListerWatcher) nacosAddress() string {
	return lw.cfg.Address.Registry
}

func (lw *NacosServiceListerWatcher) mesh() string {
	return lw.cfg.Name
}
