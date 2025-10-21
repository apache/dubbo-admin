package kubernetes

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/engine"
)

func init() {
	engine.RegisterFactory(NewKubernetesEngineFactory())
}

var _ engine.Factory = &EngineFactory{}

type EngineFactory struct{}

func NewKubernetesEngineFactory() *EngineFactory {
	return &EngineFactory{}
}

func (e *EngineFactory) Support(typ enginecfg.Type) bool {
	return enginecfg.Kubernetes == typ
}

func (e *EngineFactory) NewListWatchers(cfg *enginecfg.Config) ([]controller.ResourceListerWatcher, error) {
	kubeconfigPath := cfg.Properties.KubeConfigPath

	var config *rest.Config
	var err error
	if !strutil.IsBlank(kubeconfigPath) {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to init kubeconfig in kubernetes engine, %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to init clientset in kubernetes engine, %w", err)
	}

	lwList := make([]controller.ResourceListerWatcher, 0)
	podListerWatcher, err := NewPodListWatcher(clientset, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init PodListerWatcher in kubernetes engine, %w", err)
	}
	lwList = append(lwList, podListerWatcher)
	return lwList, nil
}
