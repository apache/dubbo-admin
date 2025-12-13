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

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/go-co-op/gocron"
	nacosconfigclient "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	nacosmodel "github.com/nacos-group/nacos-sdk-go/v2/model"
	nacosvo "github.com/nacos-group/nacos-sdk-go/v2/vo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type ConfigToResourceFunc func(mesh string, dataId string, content string) coremodel.Resource

type ConfigListerWatcher[T coremodel.Resource] struct {
	rk           coremodel.ResourceKind
	cfg          *discoverycfg.Config
	configClient nacosconfigclient.IConfigClient
	resultChan   chan watch.Event
	// watchingCfg is used to record the configs that are currently being watched
	// key-> dataId, value-> true if it is watched in current list turn
	watchingCfg    map[string]bool
	scheduler      *gocron.Scheduler
	newResListFunc coremodel.NewResourceListFunc
	toResourceFunc ConfigToResourceFunc
	blurSearch     bool
	searchExpr     string
	nacosGroup     string
	stopWatch      bool
}

func NewConfigListerWatcher(
	rk coremodel.ResourceKind,
	cfg *discoverycfg.Config,
	configClient nacosconfigclient.IConfigClient,
	toResourceFunc ConfigToResourceFunc,
	blurSearch bool,
	searchExpr string,
	nacosGroup string,
) (*ConfigListerWatcher[coremodel.Resource], error) {
	listFunc, err := coremodel.ResourceSchemaRegistry().NewResourceListFunc(rk)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.UnknownError,
			fmt.Sprintf("get resource schema failed, cause: %s", err.Error()))
	}
	return &ConfigListerWatcher[coremodel.Resource]{
		rk:             rk,
		cfg:            cfg,
		configClient:   configClient,
		resultChan:     make(chan watch.Event),
		watchingCfg:    make(map[string]bool),
		scheduler:      gocron.NewScheduler(time.UTC),
		newResListFunc: listFunc,
		toResourceFunc: toResourceFunc,
		blurSearch:     blurSearch,
		searchExpr:     searchExpr,
		nacosGroup:     nacosGroup,
		stopWatch:      false,
	}, nil
}

func (lw *ConfigListerWatcher[T]) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	resList := lw.newResListFunc()
	configs, err := lw.fetchAllConfigs()
	if err != nil {
		return nil, err
	}
	resList.SetItems(configs)
	return resList, nil
}

func (lw *ConfigListerWatcher[T]) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	_, err := lw.scheduler.Every(lw.cfg.Properties.ConfigWatchPeriod).Seconds().Do(func() {
		if lw.stopWatch {
			logger.Debugf("stop watching %s in nacos %s", lw.rk, lw.nacosAddress())
			lw.scheduler.Stop()
		}
		startTime := time.Now()
		logger.Debugf("start fetching and processing all %s in nacos %s at %s", lw.rk, lw.nacosAddress(), startTime)
		err := lw.fetchAndProcessConfigs()
		if err != nil {
			logger.Errorf("fetchAndProcessConfigs %s in nacos %s failed, cause: %s", lw.rk, lw.nacosAddress(), err.Error())
			return
		}
		costs := time.Now().UnixMilli() - startTime.UnixMilli()
		logger.Debugf("fetchAndProcessConfigs succeed %s in nacos %s, costs %dms", lw.rk, lw.nacosAddress(), costs)

	})
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.UnknownError,
			fmt.Sprintf("fetch all metadata of nacos %s in a schedule occurs error", lw.nacosAddress()))
	}
	lw.scheduler.StartAsync()
	return lw, nil
}

func (lw *ConfigListerWatcher[T]) fetchAllConfigs() ([]coremodel.Resource, error) {
	configList := make([]coremodel.Resource, 0)
	pageNum := 1
	pageSize := 100
	// list all configs by page search
	for {
		configPage, err := lw.listPage(pageNum, pageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range configPage.PageItems {
			r := lw.toResourceFunc(lw.mesh(), item.DataId, item.Content)
			if r == nil {
				logger.Warnf("config %s convert to %s failed in %s, raw content: %s", item.DataId, lw.rk, lw.nacosAddress(), item.Content)
				continue
			}
			configList = append(configList, r)
		}
		if configPage.TotalCount <= pageNum*pageSize {
			break
		}
		pageNum++
	}
	return configList, nil
}

func (lw *ConfigListerWatcher[T]) fetchAndProcessConfigs() error {
	pageNum := 1
	pageSize := 50
	// list all configs by page search
	for {
		configPage, err := lw.listPage(pageNum, pageSize)
		if err != nil {
			return err
		}
		for _, item := range configPage.PageItems {
			lw.processConfig(item)
		}
		if configPage.TotalCount <= pageNum*pageSize {
			break
		}
		pageNum++
	}
	// remove listeners for configs that are not watched in current turn
	for dataId, watched := range lw.watchingCfg {
		if watched {
			lw.watchingCfg[dataId] = false
			continue
		}
		err := lw.unsubscribeConfig(dataId, lw.nacosGroup)
		if err != nil {
			logger.Errorf("unsubscribe failed, cause: %s", err)
			continue
		}
		// only one go routine, thread safe
		delete(lw.watchingCfg, dataId)
	}
	return nil
}

func (lw *ConfigListerWatcher[T]) subscribeConfig(configKey string, group string) error {
	logger.Infof("subscribe config %s in nacos %s", configKey, lw.nacosAddress())
	err := lw.configClient.ListenConfig(nacosvo.ConfigParam{
		DataId: configKey,
		Group:  group,
		OnChange: func(namespace string, group string, dataId string, content string) {
			lw.resultChan <- watch.Event{
				Type:   watch.Modified,
				Object: lw.toResourceFunc(lw.mesh(), dataId, content),
			}
		},
	})
	if err != nil {
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("subscribe config %s failed in nacos %s", configKey, lw.nacosAddress()))
	}
	return nil
}

func (lw *ConfigListerWatcher[T]) unsubscribeConfig(configKey string, group string) error {
	logger.Infof("unsubscribe config %s in nacos %s", configKey, lw.nacosAddress())
	err := lw.configClient.CancelListenConfig(nacosvo.ConfigParam{
		DataId: configKey,
		Group:  group,
	})
	lw.resultChan <- watch.Event{
		Type:   watch.Deleted,
		Object: lw.toResourceFunc(lw.mesh(), configKey, ""),
	}
	if err != nil {
		return bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("unsubscribe config %s failed in nacos %s", configKey, lw.nacosAddress()))
	}
	return nil
}

func (lw *ConfigListerWatcher[T]) listPage(pageNum int, pageSize int) (*nacosmodel.ConfigPage, error) {
	var searchOp string
	if lw.blurSearch {
		searchOp = "blur"
	} else {
		searchOp = "accurate"
	}
	configPage, err := lw.configClient.SearchConfig(nacosvo.SearchConfigParam{
		Search:   searchOp,
		DataId:   lw.searchExpr,
		Group:    lw.nacosGroup,
		PageNo:   pageNum,
		PageSize: pageSize,
	})
	if err != nil {
		errMsg := fmt.Sprintf("cannot do page list of config, page size %d, page num %d, nacos address %s",
			pageSize, pageNum, lw.nacosAddress())
		return nil, bizerror.Wrap(err, bizerror.NacosError, errMsg)
	}
	logger.Debugf("list config page, page size %d, page num %d, nacos address %s, total count %d, items %s",
		pageSize, pageNum, lw.nacosAddress(), configPage.TotalCount, convertor.ToString(configPage.PageItems))
	return configPage, nil
}

func (lw *ConfigListerWatcher[T]) processConfig(item nacosmodel.ConfigItem) {
	// convert to resource
	res := lw.toResourceFunc(lw.mesh(), item.DataId, item.Content)
	// if resource is nil, skip the event emitting and other operations
	if res == nil {
		logger.Warnf("config %s to resource failed, raw content: %s, deleting %s in store", item.DataId, item.Content)
		return
	}
	// emit event
	lw.resultChan <- watch.Event{
		Type:   watch.Modified,
		Object: res,
	}
	// subscribe config if not watched
	if _, exists := lw.watchingCfg[item.DataId]; exists {
		lw.watchingCfg[item.DataId] = true
		return
	}
	err := lw.subscribeConfig(item.DataId, item.Group)
	if err != nil {
		logger.Errorf("subscribe config %s failed in nacos %s, cause: %v", item.DataId, lw.nacosAddress(), err)
		return
	}
	lw.watchingCfg[item.DataId] = true
}

func (lw *ConfigListerWatcher[T]) ResourceKind() coremodel.ResourceKind {
	return lw.rk
}

func (lw *ConfigListerWatcher[T]) Stop() {
	lw.stopWatch = true
}

func (lw *ConfigListerWatcher[T]) ResultChan() <-chan watch.Event {
	return lw.resultChan
}

func (lw *ConfigListerWatcher[T]) TransformFunc() cache.TransformFunc {
	return nil
}

func (lw *ConfigListerWatcher[T]) nacosAddress() string {
	return lw.cfg.Address.ConfigCenter
}

func (lw *ConfigListerWatcher[T]) mesh() string {
	return lw.cfg.Name
}
