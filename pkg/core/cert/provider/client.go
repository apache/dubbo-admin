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

package provider

import (
	"context"
	"reflect"
	"strings"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"

	admissionregistrationV1 "k8s.io/api/admissionregistration/v1"
	k8sauth "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type Client interface {
	GetAuthorityCert(namespace string) (string, string)
	UpdateAuthorityCert(cert string, pri string, namespace string)
	UpdateAuthorityPublicKey(cert string) bool
	VerifyServiceAccount(token string, authorizationType string) (*endpoint.Endpoint, bool)
	UpdateWebhookConfig(options *dubbo_cp.Config, storage *CertStorage)
	GetNamespaceLabels(namespace string) map[string]string
	GetKubClient() kubernetes.Interface
}

type ClientImpl struct {
	kubeClient kubernetes.Interface
}

func NewClient(kubeClient kubernetes.Interface) Client {
	return &ClientImpl{
		kubeClient: kubeClient,
	}
}

func (c *ClientImpl) GetAuthorityCert(namespace string) (string, string) {
	s, err := c.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), "dubbo-ca-secret", metav1.GetOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Unable to get authority cert secret from kubernetes. " + err.Error())
	}
	return string(s.Data["cert.pem"]), string(s.Data["pri.pem"])
}

func (c *ClientImpl) UpdateAuthorityCert(cert string, pri string, namespace string) {
	s, err := c.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), "dubbo-ca-secret", metav1.GetOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Unable to get ca secret from kubernetes. Will try to create. " + err.Error())
		s = &v1.Secret{
			Data: map[string][]byte{
				"cert.pem": []byte(cert),
				"pri.pem":  []byte(pri),
			},
		}
		s.Name = "dubbo-ca-secret"
		_, err = c.kubeClient.CoreV1().Secrets(namespace).Create(context.TODO(), s, metav1.CreateOptions{})
		if err != nil {
			logger.Sugar().Warnf("[Authority] Failed to create ca secret to kubernetes. " + err.Error())
		} else {
			logger.Sugar().Info("[Authority] Create ca secret to kubernetes success. ")
		}
	}

	if string(s.Data["cert.pem"]) == cert && string(s.Data["pri.pem"]) == pri {
		logger.Sugar().Info("[Authority] Ca secret in kubernetes is already the newest version.")
		return
	}

	s.Data["cert.pem"] = []byte(cert)
	s.Data["pri.pem"] = []byte(pri)
	_, err = c.kubeClient.CoreV1().Secrets(namespace).Update(context.TODO(), s, metav1.UpdateOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to update ca secret to kubernetes. " + err.Error())
	} else {
		logger.Sugar().Info("[Authority] Update ca secret to kubernetes success. ")
	}
}

func (c *ClientImpl) UpdateAuthorityPublicKey(cert string) bool {
	ns, err := c.kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to get namespaces. " + err.Error())
		return false
	}
	for _, n := range ns.Items {
		if n.Name == "kube-system" {
			continue
		}
		cm, err := c.kubeClient.CoreV1().ConfigMaps(n.Name).Get(context.TODO(), "dubbo-ca-cert", metav1.GetOptions{})
		if err != nil {
			logger.Sugar().Warnf("[Authority] Unable to find dubbo-ca-cert in " + n.Name + ". Will create config map. " + err.Error())
			cm = &v1.ConfigMap{
				Data: map[string]string{
					"ca.crt": cert,
				},
			}
			cm.Name = "dubbo-ca-cert"
			_, err = c.kubeClient.CoreV1().ConfigMaps(n.Name).Create(context.TODO(), cm, metav1.CreateOptions{})
			if err != nil {
				logger.Sugar().Warnf("[Authority] Failed to create config map for " + n.Name + ". " + err.Error())
				return false
			} else {
				logger.Sugar().Info("[Authority] Create ca config map for " + n.Name + " success.")
			}
		}
		if cm.Data["ca.crt"] == cert {
			logger.Sugar().Info("[Authority] Ignore override ca to " + n.Name + ". Cause: Already exist.")
			continue
		}
		cm.Data["ca.crt"] = cert
		_, err = c.kubeClient.CoreV1().ConfigMaps(n.Name).Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err != nil {
			logger.Sugar().Warnf("[Authority] Failed to update config map for " + n.Name + ". " + err.Error())
			return false
		} else {
			logger.Sugar().Info("[Authority] Update ca config map for " + n.Name + " success.")
		}
	}
	return true
}

func (c *ClientImpl) GetNamespaceLabels(namespace string) map[string]string {
	ns, err := c.kubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to validate token. " + err.Error())
		return map[string]string{}
	}
	if ns.Labels != nil {
		return ns.Labels
	}
	return map[string]string{}
}

func (c *ClientImpl) VerifyServiceAccount(token string, authorizationType string) (*endpoint.Endpoint, bool) {
	var tokenReview *k8sauth.TokenReview
	if authorizationType == "dubbo-ca-token" {
		tokenReview = &k8sauth.TokenReview{
			Spec: k8sauth.TokenReviewSpec{
				Token:     token,
				Audiences: []string{"dubbo-ca"},
			},
		}
	} else {
		tokenReview = &k8sauth.TokenReview{
			Spec: k8sauth.TokenReviewSpec{
				Token: token,
			},
		}
	}

	reviewRes, err := c.kubeClient.AuthenticationV1().TokenReviews().Create(
		context.TODO(), tokenReview, metav1.CreateOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to validate token. " + err.Error())
		return nil, false
	}

	if reviewRes.Status.Error != "" {
		logger.Sugar().Warnf("[Authority] Failed to validate token. " + reviewRes.Status.Error)
		return nil, false
	}

	names := strings.Split(reviewRes.Status.User.Username, ":")
	if len(names) != 4 {
		logger.Sugar().Warnf("[Authority] Token is not a pod service account. " + reviewRes.Status.User.Username)
		return nil, false
	}

	namespace := names[2]
	podName := reviewRes.Status.User.Extra["authentication.kubernetes.io/pod-name"]
	podUid := reviewRes.Status.User.Extra["authentication.kubernetes.io/pod-uid"]

	if len(podName) != 1 || len(podUid) != 1 {
		logger.Sugar().Warnf("[Authority] Token is not a pod service account. " + reviewRes.Status.User.Username)
		return nil, false
	}

	pod, err := c.kubeClient.CoreV1().Pods(namespace).Get(context.TODO(), podName[0], metav1.GetOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to get pod. " + err.Error())
		return nil, false
	}

	if pod.UID != types.UID(podUid[0]) {
		logger.Sugar().Warnf("[Authority] Token is not a pod service account. " + reviewRes.Status.User.Username)
		return nil, false
	}

	e := &endpoint.Endpoint{}

	e.ID = string(pod.UID)
	for _, i := range pod.Status.PodIPs {
		if i.IP != "" {
			e.Ips = append(e.Ips, i.IP)
		}
	}

	e.SpiffeID = "spiffe://cluster.local/ns/" + pod.Namespace + "/sa/" + pod.Spec.ServiceAccountName

	if strings.HasPrefix(reviewRes.Status.User.Username, "system:serviceaccount:") {
		names := strings.Split(reviewRes.Status.User.Username, ":")
		if len(names) == 4 {
			e.SpiffeID = "spiffe://cluster.local/ns/" + names[2] + "/sa/" + names[3]
		}
	}

	e.KubernetesEnv = &endpoint.KubernetesEnv{
		Namespace:      pod.Namespace,
		PodName:        pod.Name,
		PodLabels:      pod.Labels,
		PodAnnotations: pod.Annotations,
	}

	return e, true
}

func (c *ClientImpl) UpdateWebhookConfig(options *dubbo_cp.Config, storage *CertStorage) {
	path := "/mutating-services"
	failurePolicy := admissionregistrationV1.Ignore
	sideEffects := admissionregistrationV1.SideEffectClassNone
	bundle := storage.GetAuthorityCert().CertPem
	mwConfig, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), "dubbo-ca", metav1.GetOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Webhook] Unable to find dubbo-ca webhook config. Will create. " + err.Error())
		mwConfig = &admissionregistrationV1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dubbo-ca",
			},
			Webhooks: []admissionregistrationV1.MutatingWebhook{
				{
					Name: "dubbo-ca" + ".k8s.io",
					ClientConfig: admissionregistrationV1.WebhookClientConfig{
						Service: &admissionregistrationV1.ServiceReference{
							Name:      options.KubeConfig.ServiceName,
							Namespace: options.KubeConfig.Namespace,
							Port:      &options.Security.WebhookPort,
							Path:      &path,
						},
						CABundle: []byte(bundle),
					},
					FailurePolicy: &failurePolicy,
					Rules: []admissionregistrationV1.RuleWithOperations{
						{
							Operations: []admissionregistrationV1.OperationType{
								admissionregistrationV1.Create,
							},
							Rule: admissionregistrationV1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods"},
							},
						},
					},
					//NamespaceSelector: &metav1.LabelSelector{
					//	MatchLabels: map[string]string{
					//		"dubbo-injection": "enabled",
					//	},
					//},
					//ObjectSelector: &metav1.LabelSelector{
					//	MatchLabels: map[string]string{
					//		"dubbo-injection": "enabled",
					//	},
					//},
					SideEffects:             &sideEffects,
					AdmissionReviewVersions: []string{"v1"},
				},
			},
		}

		_, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), mwConfig, metav1.CreateOptions{})
		if err != nil {
			logger.Sugar().Warnf("[Webhook] Failed to create webhook config. " + err.Error())
		} else {
			logger.Sugar().Info("[Webhook] Create webhook config success.")
		}
		return
	}

	if reflect.DeepEqual(mwConfig.Webhooks[0].ClientConfig.CABundle, []byte(bundle)) {
		logger.Sugar().Info("[Webhook] Ignore override webhook config. Cause: Already exist.")
		return
	}

	mwConfig.Webhooks[0].ClientConfig.CABundle = []byte(bundle)
	_, err = c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.TODO(), mwConfig, metav1.UpdateOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Webhook] Failed to update webhook config. " + err.Error())
	} else {
		logger.Sugar().Info("[Webhook] Update webhook config success.")
	}
}

func (c *ClientImpl) GetKubClient() kubernetes.Interface {
	return c.kubeClient
}
