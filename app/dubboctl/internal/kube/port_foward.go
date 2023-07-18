// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForward struct {
	podName   string
	namespace string
	// localAddress and localPort form a local listening address, eg: localhost:8080
	localAddress string
	localPort    int
	// port of the target pod
	podPort int
	stopCh  chan struct{}
	cfg     *rest.Config
	client  *rest.RESTClient
}

func (pf *PortForward) Run() error {
	readyCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	// start a goroutine to process portforward
	go func() {
		for {
			select {
			case <-pf.stopCh:
				return
			default:
				// fail-fast is target pod is not running
				if err := pf.inspectPodStatus(); err != nil {
					errCh <- err
					return
				}
				fwReq := pf.client.Post().Resource("pods").Namespace(pf.namespace).Name(pf.podName).SubResource("portforward")
				kubeFw, err := pf.createKubePortForwarder(fwReq.URL(), readyCh)
				if err != nil {
					errCh <- err
					return
				}
				// for lost connection to target pod scenario, ForwardPorts would return nil.
				// so we put portforward processing in a loop. it would retry until user interrupts.
				if err = kubeFw.ForwardPorts(); err != nil {
					errCh <- err
					return
				}
				logger.CmdSugar().Infof("lost connection to %s pod", pf.podName)
			}
		}
	}()

	select {
	case <-readyCh:
		return nil
	case err := <-errCh:
		return fmt.Errorf("running portforward failed, err: %s", err)
	}
}

// inspectPodStatus check status of the target pod. If this pod is not running, fail fast.
func (pf *PortForward) inspectPodStatus() error {
	podReq := pf.client.Get().Resource("pods").Namespace(pf.namespace).Name(pf.podName)
	logger.CmdSugar().Info(podReq.URL().String())
	obj, err := podReq.Do(context.Background()).Get()
	if err != nil {
		return fmt.Errorf("get information of pod %s in %s namespace failed, err: %s", pf.podName, pf.namespace, err)
	}
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return fmt.Errorf("wanted pod but got %T", obj)
	}
	if pod.Status.Phase != v1.PodRunning {
		return fmt.Errorf("pod %s is not running. now it is %s", pf.podName, pod.Status.Phase)
	}
	return nil
}

// createKubePortForwarder makes use of kube api to create PortForwarder.
// It needs readyCh to tell PortForward that kube PortForwarder is ready.
func (pf *PortForward) createKubePortForwarder(reqUrl *url.URL, readyCh chan struct{}) (*portforward.PortForwarder, error) {
	trans, upgrader, err := spdy.RoundTripperFor(pf.cfg)
	if err != nil {
		return nil, fmt.Errorf("creating spdy RoundTripper failed, err: %s", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: trans}, "POST", reqUrl)
	fw, err := portforward.NewOnAddresses(
		dialer,
		[]string{pf.localAddress},
		[]string{fmt.Sprintf("%d:%d", pf.localPort, pf.podPort)},
		pf.stopCh,
		readyCh,
		io.Discard,
		os.Stderr,
	)
	if err != nil {
		return nil, fmt.Errorf("creating kube portforward failed, err: %s", err)
	}
	return fw, nil
}

// Stop close stopCh and free up resources
func (pf *PortForward) Stop() {
	close(pf.stopCh)
}

// Wait wait for closing stopCh which means that Stop function is the only way to trigger
func (pf *PortForward) Wait() {
	<-pf.stopCh
}

func NewPortForward(podName, namespace, localAddress string, localPort, podPort int, cfg *rest.Config) (*PortForward, error) {
	pf := &PortForward{
		podName:      podName,
		namespace:    namespace,
		localAddress: localAddress,
		localPort:    localPort,
		podPort:      podPort,
		stopCh:       make(chan struct{}),
		cfg:          cfg,
	}
	cli, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err
	}
	pf.client = cli
	return pf, nil
}
