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
	"reflect"
	"strconv"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type PodListerWatcher struct {
	cfg *enginecfg.Config
	lw  cache.ListerWatcher
}

var _ controller.ResourceListerWatcher = &PodListerWatcher{}

func NewPodListWatcher(clientset *kubernetes.Clientset, cfg *enginecfg.Config) (*PodListerWatcher, error) {
	var selector fields.Selector
	s := cfg.Properties.PodWatchSelector
	if strutil.IsBlank(s) {
		selector = fields.Everything()
	}
	selector, err := fields.ParseSelector(s)
	if err != nil {
		return nil, fmt.Errorf("parse selector %s failed: %v", s, err)
	}
	lw := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		metav1.NamespaceAll,
		selector,
	)
	return &PodListerWatcher{
		cfg: cfg,
		lw:  lw,
	}, nil
}

func (p *PodListerWatcher) List(options metav1.ListOptions) (k8sruntime.Object, error) {
	return p.lw.List(options)
}

func (p *PodListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	return p.lw.Watch(options)
}

func (p *PodListerWatcher) ResourceKind() coremodel.ResourceKind {
	return meshresource.RuntimeInstanceKind
}

func (p *PodListerWatcher) TransformFunc() cache.TransformFunc {
	return func(obj interface{}) (interface{}, error) {
		pod, ok := obj.(*v1.Pod)
		if !ok {
			objType := reflect.TypeOf(obj).Name()
			logger.Errorf("cannot transform %s to Pod", objType)
			return nil, bizerror.NewAssertionError("Pod", objType)
		}
		mainContainer := p.getMainContainer(pod)
		appName := p.getDubboAppName(pod)
		rpcPort := p.getDubboRPCPort(pod)
		var startTime string
		if pod.Status.StartTime != nil {
			startTime = pod.Status.StartTime.Format(constants.TimeFormatStr)
		}
		createTime := pod.CreationTimestamp.Format(constants.TimeFormatStr)
		readyTime := ""
		slice.ForEach(pod.Status.Conditions, func(_ int, c v1.PodCondition) {
			if c.Type == v1.PodReady && c.Status == v1.ConditionTrue {
				readyTime = c.LastTransitionTime.Format(constants.TimeFormatStr)
			}
		})
		phase := string(pod.Status.Phase)
		if pod.DeletionTimestamp != nil {
			phase = meshproto.InstanceTerminating
		}
		var workloadName string
		var workloadType string
		if len(pod.GetOwnerReferences()) > 0 {
			workloadName = pod.GetOwnerReferences()[0].Name
			workloadType = pod.GetOwnerReferences()[0].Kind
		}
		conditions := slice.Map(pod.Status.Conditions, func(_ int, c v1.PodCondition) *meshproto.Condition {
			return &meshproto.Condition{
				Type:               string(c.Type),
				Status:             string(c.Status),
				LastTransitionTime: c.LastTransitionTime.Format(constants.TimeFormatStr),
				Reason:             c.Reason,
				Message:            c.Message,
			}
		})
		var image string
		probes := make([]*meshproto.Probe, 0)
		if mainContainer != nil {
			image = mainContainer.Image
			if mainContainer.LivenessProbe != nil {
				port := p.getProbePort(mainContainer.LivenessProbe)
				probes = append(probes, &meshproto.Probe{
					Type: meshproto.LivenessProbe,
					Port: port,
				})
			}
			if mainContainer.ReadinessProbe != nil {
				port := p.getProbePort(mainContainer.ReadinessProbe)
				probes = append(probes, &meshproto.Probe{
					Type: meshproto.ReadinessProbe,
					Port: port,
				})
			}
			if mainContainer.StartupProbe != nil {
				port := p.getProbePort(mainContainer.StartupProbe)
				probes = append(probes, &meshproto.Probe{
					Type: meshproto.StartupProbe,
					Port: port,
				})
			}
		}
		res := meshresource.NewRuntimeInstanceResourceWithAttributes(pod.Name, p.getDubboMesh(pod))
		res.Spec = &meshproto.RuntimeInstance{
			Name:             pod.Name,
			Ip:               pod.Status.PodIP,
			RpcPort:          rpcPort,
			Image:            image,
			AppName:          appName,
			CreateTime:       createTime,
			StartTime:        startTime,
			ReadyTime:        readyTime,
			Phase:            phase,
			WorkloadName:     workloadName,
			WorkloadType:     workloadType,
			Node:             pod.Spec.NodeName,
			Probes:           probes,
			Conditions:       conditions,
			SourceEngine:     p.cfg.ID,
			SourceEngineType: string(p.cfg.Type),
		}
		return res, nil
	}
}

func (p *PodListerWatcher) getMainContainer(pod *v1.Pod) *v1.Container {
	containers := pod.Spec.Containers
	strategy := p.cfg.Properties.MainContainerChooseStrategy
	switch strategy.Type {
	case enginecfg.ChooseByLast:
		return &containers[len(containers)-1]
	case enginecfg.ChooseByIndex:
		return &containers[strategy.Index]
	case enginecfg.ChooseByName:
		c, ok := slice.FindBy[v1.Container](containers, func(_ int, c v1.Container) bool {
			return c.Name == strategy.Name
		})
		if !ok {
			logger.Warnf("pod %s has no container named %s, cannot retrieve the correct main container",
				pod.Name, strategy.Name)
			return nil
		}
		return &c
	case enginecfg.ChooseByAnnotation:
		value, exists := pod.Annotations[strategy.AnnotationKey]
		if !exists {
			logger.Warnf("pod %s has no annotation %s, cannot retrieve the correct main container",
				pod.Name, strategy.AnnotationKey)
			return nil
		}
		c, ok := slice.FindBy[v1.Container](containers, func(_ int, c v1.Container) bool {
			return c.Name == value
		})
		if !ok {
			logger.Warnf("pod %s has no container named %s, cannot retrieve the correct main container",
				pod.Name, value)
			return nil
		}
		return &c
	}
	return nil
}

func (p *PodListerWatcher) getDubboAppName(pod *v1.Pod) string {
	identifier := p.cfg.Properties.DubboAppIdentifier
	if identifier == nil {
		return ""
	}
	switch identifier.Type {
	case enginecfg.IdentifyByAnnotation:
		return pod.Annotations[identifier.AnnotationKey]
	case enginecfg.IdentifyByLabel:
		return pod.Labels[identifier.LabelKey]
	default:
		return ""
	}
}

func (p *PodListerWatcher) getDubboRPCPort(pod *v1.Pod) int64 {
	identifier := p.cfg.Properties.DubboRPCPortIdentifier
	if identifier == nil {
		return 0
	}
	var rpcPort int64
	var err error
	switch identifier.Type {
	case enginecfg.IdentifyByAnnotation:
		rpcPort, err = strconv.ParseInt(pod.Annotations[identifier.AnnotationKey], 10, 64)
	case enginecfg.IdentifyByLabel:
		rpcPort, err = strconv.ParseInt(pod.Labels[identifier.LabelKey], 10, 64)
	default:
		rpcPort = 0
	}
	if err != nil {
		logger.Warnf("failed to parse dubbo rpc port %s, cannot retrieve the correct dubbo rpc port",
			pod.Annotations[identifier.AnnotationKey])
	}
	return rpcPort
}

func (p *PodListerWatcher) getDubboMesh(pod *v1.Pod) string {
	identifier := p.cfg.Properties.DubboDiscoveryIdentifier
	if identifier == nil {
		return constants.DefaultMesh
	}
	var mesh string
	switch identifier.Type {
	case enginecfg.IdentifyByAnnotation:
		mesh = pod.Annotations[identifier.AnnotationKey]
	case enginecfg.IdentifyByLabel:
		mesh = pod.Labels[identifier.LabelKey]
	default:
		mesh = constants.DefaultMesh
	}
	return mesh
}

func (p *PodListerWatcher) getProbePort(probe *v1.Probe) int32 {
	if probe.TCPSocket != nil {
		return probe.TCPSocket.Port.IntVal
	} else if probe.HTTPGet != nil {
		return probe.HTTPGet.Port.IntVal
	} else if probe.GRPC != nil {
		return probe.GRPC.Port
	}
	return 0
}
