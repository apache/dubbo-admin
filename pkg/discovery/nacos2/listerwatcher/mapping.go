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
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-co-op/gocron"
	nacosconfigclient "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
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

const Mapping = "mapping"

type MappingListerWatcher struct {
	cfg          *discoverycfg.Config
	configClient nacosconfigclient.IConfigClient
	resultChan   chan watch.Event
	scheduler    *gocron.Scheduler
	stopWatch    bool
}

func NewMappingListerWatcher(
	cfg *discoverycfg.Config,
	configClient nacosconfigclient.IConfigClient,
) *MappingListerWatcher {
	return &MappingListerWatcher{
		cfg:          cfg,
		configClient: configClient,
		resultChan:   make(chan watch.Event),
		scheduler:    gocron.NewScheduler(time.UTC),
		stopWatch:    false,
	}
}

func (lw *MappingListerWatcher) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	resList := meshresource.NewServiceProviderMappingResourceList()
	mappingResources, err := lw.fetchAllMappings()
	if err != nil {
		return nil, err
	}
	resList.Items = mappingResources
	return resList, nil
}

func (lw *MappingListerWatcher) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	_, err := lw.scheduler.Every(lw.cfg.Properties.ConfigWatchPeriod).Seconds().Do(func() {
		if lw.stopWatch {
			logger.Debugf("stop watching mappings")
			lw.scheduler.Stop()
		}
		startTime := time.Now()
		logger.Debugf("start fetching all mappings in nacos %s at %s", lw.cfg.Address.MetadataReport, startTime)
		mappingResources, err := lw.fetchAllMappings()
		if err != nil {
			logger.Errorf("fetch all mappings in nacos %s failed, cause: %s", lw.cfg.Address.MetadataReport, err.Error())
			return
		}
		costs := time.Now().UnixMilli() - startTime.UnixMilli()
		logger.Debugf("fetched all mappings in nacos %s, costs %dms", lw.cfg.Address.MetadataReport, costs)
		slice.ForEach(mappingResources, func(index int, item meshresource.ServiceProviderMappingResource) {
			lw.resultChan <- watch.Event{
				Type:   watch.Modified,
				Object: &item,
			}
		})
	})
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.UnknownError,
			fmt.Sprintf("fetch all mappings of nacos %s in a schedule occurs error", lw.cfg.Address.MetadataReport))
	}
	lw.scheduler.StartAsync()
	return lw, nil
}

func (lw *MappingListerWatcher) fetchAllMappings() ([]meshresource.ServiceProviderMappingResource, error) {
	listPage := func(pageNum int, pageSize int) (*nacosmodel.ConfigPage, error) {
		configPage, err := lw.configClient.SearchConfig(nacosvo.SearchConfigParam{
			Search:   "accurate",
			Group:    Mapping,
			PageNo:   pageNum,
			PageSize: pageSize,
		})
		if err != nil {
			errMsg := fmt.Sprintf("cannot do page list of config, page size %d, page num %d, nacos address %s",
				pageSize, pageNum, lw.cfg.Address.MetadataReport)
			return nil, bizerror.Wrap(err, bizerror.NacosError, errMsg)
		}
		logger.Debugf("list config page, page size %d, page num %d, nacos address %s, total count %d, items %s",
			pageSize, pageNum, lw.cfg.Address.MetadataReport, configPage.TotalCount, convertor.ToString(configPage.PageItems))
		return configPage, nil
	}
	mappingResourceList := make([]meshresource.ServiceProviderMappingResource, 0)
	pageNum := 1
	pageSize := 50
	// list all mappings by page search
	for {
		configPage, err := listPage(pageNum, pageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range configPage.PageItems {
			appNames := strings.Split(item.Content, constants.CommaSeparator)
			mappingResourceList = append(mappingResourceList, *lw.newMappingResource(item.DataId, appNames))
		}
		if configPage.TotalCount <= pageNum*pageSize {
			break
		}
		pageNum++
	}
	return mappingResourceList, nil
}

//func (lw *MappingListerWatcher) watchSingleMapping(res meshresource.ServiceProviderMappingResource) error {
//	err := lw.configClient.Client().ListenConfig(nacosvo.ConfigParam{
//		DataId: res.Spec.ServiceName,
//		Group:  lw.cfg.Address.MetadataReport,
//		OnChange: func(_, _, dataId, data string) {
//			lw.resultChan <- watch.Event{
//				Type:   watch.Modified,
//				Object: lw.newMappingResource(res.Spec.ServiceName, strings.Split(data, constants.CommaSeparator)),
//			}
//		},
//	})
//	if err != nil {
//		return bizerror.Wrap(err, bizerror.NacosError, fmt.Sprintf("cannot watch mapping %s", res.Name))
//	}
//	return nil
//}

func (lw *MappingListerWatcher) ResourceKind() coremodel.ResourceKind {
	return meshresource.ServiceProviderMappingKind
}

func (lw *MappingListerWatcher) newMappingResource(
	serviceName string,
	appNames []string) *meshresource.ServiceProviderMappingResource {
	mappingRes := meshresource.NewServiceProviderMappingResourceWithAttributes(serviceName, lw.cfg.Name)
	mappingRes.Spec = &meshproto.ServiceProviderMapping{
		ServiceName: serviceName,
		AppNames:    appNames,
	}
	return mappingRes
}

func (lw *MappingListerWatcher) Stop() {
	lw.stopWatch = true
}

func (lw *MappingListerWatcher) ResultChan() <-chan watch.Event {
	return lw.resultChan
}

func (lw *MappingListerWatcher) TransformFunc() cache.TransformFunc {
	return nil
}
