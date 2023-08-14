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

package subcmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/kube"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/operator"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// todo: think about a efficient way to change selectors and ports when yaml files change
	// ports are coming from /deploy/charts and /deploy/kubernetes
	ComponentPortMap = map[operator.ComponentName]int{
		operator.Admin:      8080,
		operator.Grafana:    3000,
		operator.Nacos:      8848,
		operator.Prometheus: 9090,
		operator.Skywalking: 8080,
		operator.Zipkin:     9411,
	}
	// selectors are coming from /deploy/charts and /deploy/kubernetes
	ComponentSelectorMap = map[operator.ComponentName]string{
		operator.Admin:      "app.kubernetes.io/name=dubbo-admin",
		operator.Grafana:    "app.kubernetes.io/name=grafana",
		operator.Nacos:      "app.kubernetes.io/name=nacos",
		operator.Prometheus: "app=prometheus",
		operator.Skywalking: "app=skywalking, component=ui",
		operator.Zipkin:     "app.kubernetes.io/name=zipkin",
	}
)

type DashboardCommonArgs struct {
	port           int
	host           string
	openBrowser    bool
	namespace      string
	KubeConfigPath string
	// selected cluster info of kubeconfig
	Context string
}

func (dca *DashboardCommonArgs) setDefault() {
	if dca == nil {
		return
	}
	if dca.host == "" {
		dca.host = "127.0.0.1"
	}
	if dca.namespace == "" {
		dca.namespace = identifier.DubboSystemNamespace
	}
}

func commonDashboardCmd(baseCmd *cobra.Command, compName operator.ComponentName) {
	nameStr := string(compName)
	lowerNameStr := strings.ToLower(nameStr)
	dcArgs := &DashboardCommonArgs{}
	cmd := &cobra.Command{
		Use:   lowerNameStr,
		Short: fmt.Sprintf("create PortForward between local address and target component %s pod. open browser by default", nameStr),
		Example: fmt.Sprintf(`  # create PortForward in 127.0.0.1:%d and open browser directly
  dubboctl dashboard %s
  # specify port
  dubboctl dashboard %s --port %d
  # do not open browser
  dubboctl dashboard %s --openBrowser false
  # specify namespace of Admin
  dubboctl dashboard %s --namespace ns_user_specified
`, ComponentPortMap[compName], lowerNameStr, lowerNameStr, ComponentPortMap[compName], lowerNameStr, lowerNameStr),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			dcArgs.setDefault()
			if err := portForward(dcArgs, compName, cmd.OutOrStdout()); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.PersistentFlags().IntVarP(&dcArgs.port, "port", "p", 0,
		fmt.Sprintf("local port to listen on. If not set, it would be same as the default port of component %s", nameStr))
	cmd.PersistentFlags().StringVarP(&dcArgs.host, "host", "h", "",
		"local host to bind. If not set, it would be 127.0.0.1")
	// openBrowser is default behaviour
	cmd.PersistentFlags().BoolVarP(&dcArgs.openBrowser, "openBrowser", "", true,
		"whether to open browser automatically")
	cmd.PersistentFlags().StringVarP(&dcArgs.namespace, "namespace", "n", "",
		fmt.Sprintf("namespace in which component %s is located", nameStr))
	cmd.PersistentFlags().StringVarP(&dcArgs.KubeConfigPath, "kubeConfig", "", "",
		"Path to kubeconfig")
	cmd.PersistentFlags().StringVarP(&dcArgs.Context, "context", "", "",
		"Context in kubeconfig to use")

	baseCmd.AddCommand(cmd)
}

func ConfigDashboardAdminCmd(baseCmd *cobra.Command) {
	commonDashboardCmd(baseCmd, operator.Admin)
}

func ConfigDashboardGrafanaCmd(baseCmd *cobra.Command) {
	commonDashboardCmd(baseCmd, operator.Grafana)
}

func ConfigDashboardNacosCmd(baseCmd *cobra.Command) {
	commonDashboardCmd(baseCmd, operator.Nacos)
}

func ConfigDashboardPrometheusCmd(baseCmd *cobra.Command) {
	commonDashboardCmd(baseCmd, operator.Prometheus)
}

func ConfigDashboardSkywalkingCmd(baseCmd *cobra.Command) {
	commonDashboardCmd(baseCmd, operator.Skywalking)
}

func ConfigDashboardZipkinCmd(baseCmd *cobra.Command) {
	commonDashboardCmd(baseCmd, operator.Zipkin)
}

func portForward(args *DashboardCommonArgs, compName operator.ComponentName, writer io.Writer) error {
	// process args
	var podPort int
	podPort = ComponentPortMap[compName]
	if args.port == 0 {
		args.port = podPort
	}

	// prepare PortForward args
	labelSelector := ComponentSelectorMap[compName]
	cfg, err := kube.BuildConfig(args.KubeConfigPath, args.Context)
	if err != nil {
		return fmt.Errorf("build kube config failed, err: %s", err)
	}
	// todo: unify kube client
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("create kube RESTClient failed, err: %s", err)
	}
	pods, err := cli.CoreV1().Pods(args.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("list pod failed, err: %s", err)
	}
	if len(pods.Items) < 1 {
		return fmt.Errorf("no %s pods found", string(compName))
	}
	// use name of the first pod
	podName := pods.Items[0].Name

	pf, err := kube.NewPortForward(podName, args.namespace, args.host, args.port, podPort, cfg)
	if err != nil {
		return fmt.Errorf("create PortForward failed, err: %s", err)
	}
	if err := pf.Run(); err != nil {
		pf.Stop()
		return fmt.Errorf("PortForward running failed, err: %s", err)
	}

	logger.CmdSugar().Infof("PortForward to %s pod is running", podName)

	// wait for interrupt
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		defer signal.Stop(signals)
		<-signals
		logger.CmdSugar().Info("PortForward stops")
		pf.Stop()
	}()

	if args.openBrowser {
		address := net.JoinHostPort(args.host, strconv.Itoa(args.port))
		url := "http://" + address
		openBrowser(url, writer)
	}

	pf.Wait()

	return nil
}

// openBrowser uses syscall from different runtime
func openBrowser(url string, writer io.Writer) {
	var err error

	fmt.Fprintf(writer, "open browser in %s\n", url)

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		fmt.Fprintf(writer, "unsupported platform %q. pls open %s in your browser.\n", runtime.GOOS, url)
	}

	if err != nil {
		fmt.Fprintf(writer, "open browser failed; open %s in your browser.\n", url)
	}
}
