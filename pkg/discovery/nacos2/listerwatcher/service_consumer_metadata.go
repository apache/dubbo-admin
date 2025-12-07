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
	"regexp"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	set "github.com/duke-git/lancet/v2/datastructure/set"
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
	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type ServiceConsumerMetadataListerWatcher struct {
	cfg               *discoverycfg.Config
	namingClient      nacosnamingclient.INamingClient
	watchingConsumers set.Set[string]
	scheduler         *gocron.Scheduler
	resultChan        chan watch.Event
	stopWatch         bool
}

func NewServiceConsumerMetadataListerWatcher(
	cfg *discoverycfg.Config,
	namingClient nacosnamingclient.INamingClient) *ServiceConsumerMetadataListerWatcher {
	return &ServiceConsumerMetadataListerWatcher{
		cfg:               cfg,
		namingClient:      namingClient,
		resultChan:        make(chan watch.Event),
		watchingConsumers: set.New[string](),
		scheduler:         gocron.NewScheduler(time.UTC),
		stopWatch:         false,
	}
}

func (lw *ServiceConsumerMetadataListerWatcher) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	consumerKeys, err := lw.fetchAllServiceConsumerKeys()
	if err != nil {
		return nil, err
	}
	resListObj := meshresource.NewServiceConsumerMetadataResourceList()
	resList := make([]meshresource.ServiceConsumerMetadataResource, 0)
	for _, ck := range consumerKeys {
		consumers, err := lw.namingClient.GetService(nacosvo.GetServiceParam{
			ServiceName: ck,
		})
		if err != nil {
			logger.Errorf("get service consumers %s failed in nacos %s, cause: %v", ck, lw.cfg.Address.Registry, err)
			continue
		}
		for _, consumer := range consumers.Hosts {
			res := lw.toServiceConsumerMetadataResource(ck, consumer)
			if res != nil {
				resList = append(resList, *res)
			}
		}
	}
	resListObj.Items = resList
	return resListObj, nil
}

func (lw *ServiceConsumerMetadataListerWatcher) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	_, err := lw.scheduler.Every(lw.cfg.Properties.ConfigWatchPeriod).Seconds().Do(func() {
		if lw.stopWatch {
			logger.Debugf("stop watch all service consumer metadata of nacos %s", lw.cfg.Address)
			lw.scheduler.Stop()
			return
		}
		startTime := time.Now()
		logger.Debugf("start fetching all service consumer metadata in nacos %s at %s", lw.cfg.Address.Registry, startTime)
		serviceConsumerNames, err := lw.fetchAllServiceConsumerKeys()
		if err != nil {
			logger.Errorf("fetch all service consumer failed in nacos %s, cause: %v", lw.cfg.Address.Registry, err)
			return
		}
		costs := time.Now().UnixMilli() - startTime.UnixMilli()
		logger.Debugf("fetch all service consumer metadata in nacos %s succeed, costs %dms", lw.cfg.Address.Registry, costs)
		newConsumers := set.FromSlice(serviceConsumerNames)
		offlineConsumers := lw.watchingConsumers.Minus(newConsumers)
		for _, consumerName := range offlineConsumers.ToSlice() {
			err := lw.unsubscribeServiceConsumer(consumerName)
			if err != nil {
				logger.Errorf("unsubscribe service consumer %s failed in nacos %s, cause: %v",
					consumerName, lw.cfg.Address.Registry, err)
			}
			lw.watchingConsumers.Delete(consumerName)
			logger.Debugf("unsubscribe service consumer %s succeed in nacos %s", consumerName, lw.cfg.Address.Registry)
			continue
		}
		for _, consumerName := range newConsumers.ToSlice() {
			exist := lw.watchingConsumers.AddIfNotExist(consumerName)
			if exist {
				continue
			}
			err := lw.subscribeServiceConsumer(consumerName)
			if err != nil {
				logger.Errorf("subscribe service consumer %s failed in nacos %s, cause: %v", consumerName, lw.cfg.Address.Registry, err)
			}
		}
	})
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.UnknownError,
			fmt.Sprintf("watch all mappings of nacos %s in a schedule occurs error", lw.cfg.Address))
	}
	lw.scheduler.StartAsync()
	return lw, nil
}

func (lw *ServiceConsumerMetadataListerWatcher) fetchAllServiceConsumerKeys() ([]string, error) {
	allKeys := make([]string, 0)
	//Filter out interface-level DataIds
	const pattern = `^consumers:[\w\.]+(?::[\w\.]*:|::[\w\.]*)?$`
	re := regexp.MustCompile(pattern)
	var pageSize uint32 = 100
	for pageNum := uint32(1); ; pageNum++ {
		consumerKeys, err := lw.namingClient.GetAllServicesInfo(nacosvo.GetAllServiceInfoParam{
			PageSize: pageSize,
			PageNo:   pageNum,
		})

		if err != nil {
			return nil, bizerror.Wrap(err, bizerror.NacosError,
				fmt.Sprintf("get service consumer keys failed in nacos %s failed, page size: %d, page num: %d",
					lw.cfg.Address, pageSize, pageNum))
		}
		for _, key := range consumerKeys.Doms {
			if re.MatchString(key) {
				allKeys = append(allKeys, key)
			}
		}
		// If the number of consumerKeys is less than the page size, it means that the last page has been reached
		if consumerKeys.Count < int64(pageSize) {
			return allKeys, nil
		}
	}
}

func (lw *ServiceConsumerMetadataListerWatcher) subscribeServiceConsumer(consumerKey string) error {
	err := lw.namingClient.Subscribe(&nacosvo.SubscribeParam{
		ServiceName: consumerKey,
		SubscribeCallback: func(instances []nacosmodel.Instance, err error) {
			if err != nil {
				logger.Errorf("subscribe service consumer %s failed in nacos %s, cause: %v",
					consumerKey, lw.cfg.Address.Registry, err)
				return
			}
			slice.ForEach(instances, func(index int, instance nacosmodel.Instance) {
				lw.resultChan <- watch.Event{
					Type: watch.Modified, Object: lw.toServiceConsumerMetadataResource(consumerKey, instance)}
			})
		},
	})
	if err != nil {
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("subscribe service consumer %s failed in nacos %s", consumerKey, lw.cfg.Address.Registry))
	}
	return nil
}

func (lw *ServiceConsumerMetadataListerWatcher) unsubscribeServiceConsumer(appName string) error {
	return lw.namingClient.Unsubscribe(&nacosvo.SubscribeParam{
		ServiceName: appName,
		SubscribeCallback: func(instances []nacosmodel.Instance, err error) {
			if err != nil {
				logger.Errorf("unsubscribe app %s failed in nacos %s, cause: %v", appName, lw.cfg.Address.Registry, err)
				return
			}
		},
	})
}

func (lw *ServiceConsumerMetadataListerWatcher) toServiceConsumerMetadataResource(consumerKey string, instance nacosmodel.Instance) *meshresource.ServiceConsumerMetadataResource {
	metadataJsonStr, err := convertor.ToJson(instance.Metadata)
	if err != nil {
		logger.Errorf("skipping convert service consumer %s:%d of %s to ServiceConsumerMetadata "+
			"because it cannot convert to json string", instance.Ip, instance.Port, consumerKey)
		return nil
	}
	consumerAppName, exists := instance.Metadata[constants.Application]
	if !exists {
		logger.Errorf("service consumer %s:%d of %s is invalid because no application name found, raw metadata: %s",
			instance.Ip, instance.Port, consumerKey, metadataJsonStr)
		return nil
	}
	serviceName, exists := instance.Metadata[constants.InterfaceKey]
	if !exists {
		logger.Errorf("service consumer %s:%d of %s is invalid because no service name found, raw metadata: %s",
			instance.Ip, instance.Port, consumerKey, metadataJsonStr)
		return nil
	}
	version := instance.Metadata[constants.VersionKey]
	group := instance.Metadata[constants.GroupKey]
	resKey := consumerAppName + ":" + consumerKey
	res := meshresource.NewServiceConsumerMetadataResourceWithAttributes(resKey, lw.cfg.Name)
	res.Spec = &meshproto.ServiceConsumerMetadata{
		ServiceName:     serviceName,
		ConsumerAppName: consumerAppName,
		Version:         version,
		Group:           group,
		Metadata:        instance.Metadata,
	}
	return res
}

func (lw *ServiceConsumerMetadataListerWatcher) ResourceKind() coremodel.ResourceKind {
	return meshresource.ServiceConsumerMetadataKind
}

func (lw *ServiceConsumerMetadataListerWatcher) TransformFunc() cache.TransformFunc {
	return nil
}

func (lw *ServiceConsumerMetadataListerWatcher) Stop() {
	lw.stopWatch = true
}

func (lw *ServiceConsumerMetadataListerWatcher) ResultChan() <-chan watch.Event {
	return lw.resultChan
}
