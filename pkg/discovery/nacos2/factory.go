package dubbogo

import (
	"fmt"

	dubbogocom "dubbo.apache.org/dubbo-go/v3/common"
	dubbogoconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	dubbogonacos "dubbo.apache.org/dubbo-go/v3/remoting/nacos"
	nacosconfigclient "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	nacosnamingclient "github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/discovery"
	"github.com/apache/dubbo-admin/pkg/discovery/nacos2/listerwatcher"
)

func init() {
	discovery.RegisterListWatcherFactory(&Factory{})
}

type Factory struct{}

func (f *Factory) Support(d discoverycfg.Type) bool {
	return d == discoverycfg.Nacos2
}

func (f *Factory) NewListWatchers(cfg *discoverycfg.Config) ([]controller.ResourceListerWatcher, error) {
	nacosConfigClient, namingClient, err := f.initNacosClients(cfg)
	if err != nil {
		return nil, err
	}
	mappingLW := listerwatcher.NewMappingListerWatcher(cfg, nacosConfigClient)
	serviceProviderMetadataLW := listerwatcher.NewServiceProviderMetadataListerWatcher(cfg, nacosConfigClient)
	serviceConsumerMetadataLW := listerwatcher.NewServiceConsumerMetadataListerWatcher(cfg, namingClient)
	rpcInstanceLW := listerwatcher.NewRPCInstanceListerWatcher(cfg, namingClient)

	return []controller.ResourceListerWatcher{
		mappingLW,
		serviceProviderMetadataLW,
		serviceConsumerMetadataLW,
		rpcInstanceLW,
	}, nil
}

func (f *Factory) initNacosClients(
	cfg *discoverycfg.Config,
) (nacosconfigclient.IConfigClient, nacosnamingclient.INamingClient, error) {
	cfgCenterUrl, err := dubbogocom.NewURL(cfg.Address.ConfigCenter)
	if err != nil {
		return nil, nil, err
	}
	cfgCenterUrl.AddParam(dubbogoconstant.ClientNameKey, cfg.Name)
	nacosConfigClient, err := dubbogonacos.NewNacosConfigClientByUrl(cfgCenterUrl)
	if err != nil {
		return nil, nil, bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("cannot create nacos config client for %s %s", cfg.Name, cfg.Address))
	}
	registryUrl, err := dubbogocom.NewURL(cfg.Address.Registry)
	if err != nil {
		return nil, nil, err
	}
	registryUrl.AddParam(dubbogoconstant.ClientNameKey, cfg.Name)
	namingClient, err := dubbogonacos.NewNacosClientByURL(registryUrl)
	if err != nil {
		return nil, nil, bizerror.Wrap(err, bizerror.NacosError,
			fmt.Sprintf("cannot create nacos naming client for %s %s", cfg.Name, cfg.Address))
	}
	return nacosConfigClient.Client(), namingClient.Client(), nil
}
