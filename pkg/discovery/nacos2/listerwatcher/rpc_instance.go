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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	set "github.com/duke-git/lancet/v2/datastructure/set"
	"github.com/duke-git/lancet/v2/datetime"
	"github.com/duke-git/lancet/v2/maputil"
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

type RPCInstanceListerWatcher struct {
	cfg          *discoverycfg.Config
	namingClient nacosnamingclient.INamingClient
	watchingApps set.Set[string]
	scheduler    *gocron.Scheduler
	resultChan   chan watch.Event
	stopWatch    bool
}

func NewRPCInstanceListerWatcher(cfg *discoverycfg.Config, namingClient nacosnamingclient.INamingClient) *RPCInstanceListerWatcher {
	return &RPCInstanceListerWatcher{
		cfg:          cfg,
		namingClient: namingClient,
		resultChan:   make(chan watch.Event),
		watchingApps: set.New[string](),
		scheduler:    gocron.NewScheduler(time.UTC),
		stopWatch:    false,
	}
}

func (lw *RPCInstanceListerWatcher) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	appNames, err := lw.fetchAllAppNames()
	if err != nil {
		return nil, err
	}
	resourceListObj := meshresource.NewRPCInstanceResourceList()
	resList := make([]meshresource.RPCInstanceResource, 0)
	for _, appName := range appNames {
		appInstances, err := lw.namingClient.GetService(nacosvo.GetServiceParam{
			ServiceName: appName,
		})
		if err != nil {
			logger.Errorf("get instances of app %s failed in nacos %s, cause: %v", appName, lw.cfg.Address.Registry, err)
			continue
		}
		for _, instance := range appInstances.Hosts {
			resList = append(resList, *lw.toRPCInstanceResource(appName, instance))
		}
	}
	resourceListObj.Items = resList
	return resourceListObj, nil
}

func (lw *RPCInstanceListerWatcher) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	_, err := lw.scheduler.Every(lw.cfg.Properties.ServiceWatchPeriod).Seconds().Do(func() {
		if lw.stopWatch {
			logger.Debugf("stop watch all rpc instances of nacos %s", lw.cfg.Address.Registry)
			lw.scheduler.Stop()
			return
		}
		startTime := time.Now()
		logger.Debugf("start fetching all rpc instances in nacos %s at %s", lw.cfg.Address.Registry, startTime)
		appNames, err := lw.fetchAllAppNames()
		if err != nil {
			logger.Errorf("fetch all app names failed in nacos %s, cause: %v", lw.cfg.Address.Registry, err)
			return
		}
		costs := time.Now().UnixMilli() - startTime.UnixMilli()
		logger.Debugf("finish fetching all rpc instances in nacos %s, costs %dms", lw.cfg.Address.Registry, costs)
		newAppNames := set.FromSlice(appNames)
		offlineAppNames := lw.watchingApps.Minus(newAppNames)
		for _, appName := range offlineAppNames.ToSlice() {
			err := lw.unsubscribeApp(appName)
			if err != nil {
				logger.Errorf("unsubscribe app %s failed in nacos %s, cause: %v", appName, lw.cfg.Address.Registry, err)
			}
			lw.watchingApps.Delete(appName)
			continue
		}
		for _, appName := range newAppNames.ToSlice() {
			exist := lw.watchingApps.AddIfNotExist(appName)
			if exist {
				continue
			}
			err := lw.subscribeApp(appName)
			if err != nil {
				logger.Errorf("subscribe app %s failed in nacos %s, cause: %v", appName, lw.cfg.Address.Registry, err)
			}
		}
	})
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.UnknownError,
			fmt.Sprintf("watch all mappings of nacos %s in a schedule occurs error", lw.cfg.Address.Registry))
	}
	lw.scheduler.StartAsync()
	return lw, nil
}

func (lw *RPCInstanceListerWatcher) fetchAllAppNames() ([]string, error) {
	appNames := make([]string, 0)
	// filter out interface-level provider metadata dataIds
	const providerPattern = `^providers:[\w\.]+(?::[\w\.]*:|::[\w\.]*)?$`
	providerRe := regexp.MustCompile(providerPattern)
	// filter out consumer metadata dataIds
	const consumerPattern = `^consumers:[\w\.]+(?::[\w\.]*:|::[\w\.]*)?$`
	consumerRe := regexp.MustCompile(consumerPattern)
	var pageSize uint32 = 100
	for pageNum := uint32(1); ; pageNum++ {
		appInfos, err := lw.namingClient.GetAllServicesInfo(nacosvo.GetAllServiceInfoParam{
			PageSize: pageSize,
			PageNo:   pageNum,
		})

		if err != nil {
			return nil, bizerror.Wrap(err, bizerror.NacosError,
				fmt.Sprintf("get app names failed in nacos %s failed, page size: %d, page num: %d", lw.cfg.Address.Registry, pageSize, pageNum))
		}
		for _, e := range appInfos.Doms {
			if !providerRe.MatchString(e) && !consumerRe.MatchString(e) {
				appNames = append(appNames, e)
			}
		}
		// If the number of appInfos is less than the page size, it means that the last page has been reached
		if appInfos.Count < int64(pageSize) {
			return appNames, nil
		}
	}
}

func (lw *RPCInstanceListerWatcher) subscribeApp(appName string) error {
	err := lw.namingClient.Subscribe(&nacosvo.SubscribeParam{
		ServiceName: appName,
		SubscribeCallback: func(instances []nacosmodel.Instance, err error) {
			if err != nil {
				logger.Errorf("subscribe app %s failed in nacos %s, cause: %v", appName, lw.cfg.Address.Registry, err)
				return
			}
			slice.ForEach(instances, func(index int, instance nacosmodel.Instance) {
				lw.resultChan <- watch.Event{
					Type: watch.Deleted, Object: lw.toRPCInstanceResource(appName, instance)}
			})
		},
	})
	if err != nil {
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("subscribe app %s failed in nacos %s", appName, lw.cfg.Address.Registry))
	}
	lw.watchingApps.Add(appName)
	return nil
}

func (lw *RPCInstanceListerWatcher) unsubscribeApp(appName string) error {
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

func (lw *RPCInstanceListerWatcher) toRPCInstanceResource(appName string, instance nacosmodel.Instance) *meshresource.RPCInstanceResource {
	resName := meshresource.BuildInstanceResName(appName, instance.Ip, int64(instance.Port))
	res := meshresource.NewRPCInstanceResourceWithAttributes(resName, lw.cfg.Name)
	var registerTime string
	timestamp, err := strconv.ParseInt(instance.Metadata[constants.TimestampKey], 10, 64)
	if err != nil {
		registerTime = ""
	} else {
		registerTime = datetime.FormatTimeToStr(time.UnixMilli(timestamp), "2006-01-02 15:04:05")
	}
	revision := instance.Metadata[constants.MetadataRevisionKey]
	metadataStorageType := instance.Metadata[constants.MetadataStorageTypeKey]
	urlParams, exists := instance.Metadata[constants.URLParamsKey]
	relaseVersion := ""
	protocol := ""
	serialization := ""
	preferSerialization := ""
	if exists {
		paramsMap := make(map[string]string)
		err := json.Unmarshal([]byte(urlParams), &paramsMap)
		if err != nil {
			logger.Warnf("parse url params failed, raw url params string: %s, cause: %v", urlParams, err)
		}
		relaseVersion = paramsMap[constants.ReleaseKey]
		protocol = paramsMap[constants.DubboVersionKey]
		serialization = paramsMap[constants.SerializationKey]
		preferSerialization = paramsMap[constants.SerializationKey]

	}
	var endpoints []*meshproto.Endpoint
	err = json.Unmarshal([]byte(instance.Metadata[constants.EndpointsKey]), &endpoints)
	if err != nil {
		logger.Warnf("parse endpoints failed, raw endpoints string: %s, cause: %v", instance.Metadata[constants.EndpointsKey], err)
	}
	res.Spec = &meshproto.RPCInstance{
		Name:                resName,
		AppName:             appName,
		Ip:                  instance.Ip,
		Port:                int64(instance.Port),
		RegisterTime:        registerTime,
		UnregisterTime:      "",
		Revision:            revision,
		MetadataStorageType: metadataStorageType,
		ReleaseVersion:      relaseVersion,
		Protocol:            protocol,
		Serialization:       serialization,
		PreferSerialization: preferSerialization,
		Tags:                getRPCInstanceTags(instance.Metadata),
		Endpoints:           endpoints,
		Metadata:            instance.Metadata,
	}
	return res
}

func getRPCInstanceTags(metadata map[string]string) map[string]string {
	knownKeys := set.New[string](constants.URLParamsKey, constants.EndpointsKey,
		constants.MetadataRevisionKey, constants.MetadataStorageTypeKey, constants.TimestampKey)
	return maputil.Filter(metadata, func(key string, value string) bool {
		return !knownKeys.Contain(key)
	})
}

func (lw *RPCInstanceListerWatcher) ResourceKind() coremodel.ResourceKind {
	return meshresource.RPCInstanceKind
}

func (lw *RPCInstanceListerWatcher) TransformFunc() cache.TransformFunc {
	return nil
}

func (lw *RPCInstanceListerWatcher) Stop() {
	lw.stopWatch = true
}

func (lw *RPCInstanceListerWatcher) ResultChan() <-chan watch.Event {
	return lw.resultChan
}
