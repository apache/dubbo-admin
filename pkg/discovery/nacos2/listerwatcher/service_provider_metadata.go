package listerwatcher

import (
	"encoding/json"
	"fmt"
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

const DubboGroup = "dubbo"

type ServiceProviderMetadataListerWatcher struct {
	cfg          *discoverycfg.Config
	configClient nacosconfigclient.IConfigClient
	resultChan   chan watch.Event
	scheduler    *gocron.Scheduler
	stopWatch    bool
}

func NewServiceProviderMetadataListerWatcher(
	cfg *discoverycfg.Config,
	configClient nacosconfigclient.IConfigClient,
) *ServiceProviderMetadataListerWatcher {
	return &ServiceProviderMetadataListerWatcher{
		cfg:          cfg,
		configClient: configClient,
		resultChan:   make(chan watch.Event),
		scheduler:    gocron.NewScheduler(time.UTC),
		stopWatch:    false,
	}
}

func (lw *ServiceProviderMetadataListerWatcher) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	resList := meshresource.NewServiceProviderMetadataResourceList()
	metadataResources, err := lw.fetchAllProviderMetadata()
	if err != nil {
		return nil, err
	}
	resList.Items = metadataResources
	return resList, nil
}

func (lw *ServiceProviderMetadataListerWatcher) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	_, err := lw.scheduler.Every(lw.cfg.Properties.ConfigWatchPeriod).Seconds().Do(func() {
		if lw.stopWatch {
			logger.Debugf("stop watching metadata in nacos %s", lw.cfg.Address.MetadataReport)
			lw.scheduler.Stop()
		}
		startTime := time.Now()
		logger.Debugf("start fetching all service provider metadata in nacos %s at %s", lw.cfg.Address.MetadataReport, startTime)
		metadataResources, err := lw.fetchAllProviderMetadata()
		if err != nil {
			logger.Errorf("fetch all service provider metadata in nacos %s failed, cause: %s", lw.cfg.Address.MetadataReport, err.Error())
			return
		}
		costs := time.Now().UnixMilli() - startTime.UnixMilli()
		logger.Debugf("fetched all service provider metadata in nacos %s, costs %dms", lw.cfg.Address.MetadataReport, costs)
		slice.ForEach(metadataResources, func(index int, item meshresource.ServiceProviderMetadataResource) {
			lw.resultChan <- watch.Event{
				Type:   watch.Modified,
				Object: &item,
			}
		})
	})
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.UnknownError,
			fmt.Sprintf("fetch all metadata of nacos %s in a schedule occurs error", lw.cfg.Address.MetadataReport))
	}
	lw.scheduler.StartAsync()
	return lw, nil
}

func (lw *ServiceProviderMetadataListerWatcher) fetchAllProviderMetadata() ([]meshresource.ServiceProviderMetadataResource, error) {
	listPage := func(pageNum int, pageSize int) (*nacosmodel.ConfigPage, error) {
		configPage, err := lw.configClient.SearchConfig(nacosvo.SearchConfigParam{
			Search:   "blur",
			DataId:   "*:provider:*",
			Group:    DubboGroup,
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
	resList := make([]meshresource.ServiceProviderMetadataResource, 0)
	pageNum := 1
	pageSize := 50
	// list all metadata by page search
	for {
		configPage, err := listPage(pageNum, pageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range configPage.PageItems {
			res := lw.toMetadataResource(item)
			if res != nil {
				resList = append(resList, *res)
			}
		}
		if configPage.TotalCount <= pageNum*pageSize {
			break
		}
		pageNum++
	}
	return resList, nil
}

func (lw *ServiceProviderMetadataListerWatcher) ResourceKind() coremodel.ResourceKind {
	return meshresource.ServiceProviderMetadataKind
}

func (lw *ServiceProviderMetadataListerWatcher) toMetadataResource(configItem nacosmodel.ConfigItem) *meshresource.ServiceProviderMetadataResource {
	metadataSpec := &meshproto.ServiceProviderMetadata{}
	err := json.Unmarshal([]byte(configItem.Content), metadataSpec)
	if err != nil {
		logger.Errorf("failed to unmarshal provider service metadata %s, cause: %v", configItem.Content, err)
		return nil
	}
	metadataSpec.ServiceName = metadataSpec.CanonicalName
	metadataSpec.Version = metadataSpec.Parameters[constants.VersionKey]
	metadataSpec.Group = metadataSpec.Parameters[constants.GroupKey]

	metadataRes := meshresource.NewServiceProviderMetadataResourceWithAttributes(configItem.DataId, lw.cfg.Name)
	metadataRes.Spec = metadataSpec
	return metadataRes
}

func (lw *ServiceProviderMetadataListerWatcher) Stop() {
	lw.stopWatch = true
}

func (lw *ServiceProviderMetadataListerWatcher) ResultChan() <-chan watch.Event {
	return lw.resultChan
}

func (lw *ServiceProviderMetadataListerWatcher) TransformFunc() cache.TransformFunc {
	return nil
}
