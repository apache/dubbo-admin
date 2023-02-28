package patch

import (
	"github.com/apache/dubbo-admin/ca/pkg/config"
	"github.com/apache/dubbo-admin/ca/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	"strconv"
)

type JavaSdk struct {
	options    *config.Options
	kubeClient k8s.Client
}

func NewJavaSdk(options *config.Options) *JavaSdk {
	return &JavaSdk{
		options: options,
	}
}

func (s *JavaSdk) NewPod(origin *v1.Pod) (*v1.Pod, error) {
	target := origin.DeepCopy()
	expireSeconds := int64(3600)

	shouldInject := false

	if target.Labels["dubbo-ca.inject"] == "true" {
		shouldInject = true
	}

	if !shouldInject && s.kubeClient.GetNamespaceLabels(target.Namespace)["dubbo-ca.inject"] == "true" {
		shouldInject = true
	}

	if shouldInject {
		for _, v := range target.Spec.Volumes {
			if v.Name == "dubbo-ca-token" {
				shouldInject = false
				break
			}
		}
		for _, v := range target.Spec.Volumes {
			if v.Name == "dubbo-ca-cert" {
				shouldInject = false
				break
			}
		}

		for _, c := range target.Spec.Containers {
			for _, e := range c.Env {
				if e.Name == "DUBBO_CA_ADDRESS" {
					shouldInject = false
					break
				}
				if e.Name == "DUBBO_CA_CERT_PATH" {
					shouldInject = false
					break
				}
				if e.Name == "DUBBO_OIDC_TOKEN" {
					shouldInject = false
					break
				}
			}

			for _, m := range c.VolumeMounts {
				if m.Name == "dubbo-ca-token" {
					shouldInject = false
					break
				}
				if m.Name == "dubbo-ca-cert" {
					shouldInject = false
					break
				}
			}
		}
	}

	if shouldInject {
		target.Spec.Volumes = append(target.Spec.Volumes, v1.Volume{
			Name: "dubbo-ca-token",
			VolumeSource: v1.VolumeSource{
				Projected: &v1.ProjectedVolumeSource{
					Sources: []v1.VolumeProjection{
						{
							ServiceAccountToken: &v1.ServiceAccountTokenProjection{
								Audience:          "dubbo-ca",
								ExpirationSeconds: &expireSeconds,
								Path:              "token",
							},
						},
					},
				},
			},
		})
		target.Spec.Volumes = append(target.Spec.Volumes, v1.Volume{
			Name: "dubbo-ca-cert",
			VolumeSource: v1.VolumeSource{
				Projected: &v1.ProjectedVolumeSource{
					Sources: []v1.VolumeProjection{
						{
							ConfigMap: &v1.ConfigMapProjection{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "dubbo-ca-cert",
								},
								Items: []v1.KeyToPath{
									{
										Key:  "ca.crt",
										Path: "ca.crt",
									},
								},
							},
						},
					},
				},
			},
		})

		targetContainers := []v1.Container{}
		for _, c := range target.Spec.Containers {
			c.Env = append(c.Env, v1.EnvVar{
				Name:  "DUBBO_CA_ADDRESS",
				Value: s.options.ServiceName + "." + s.options.Namespace + ".svc:" + strconv.Itoa(int(s.options.SecureServerPort)),
			})
			c.Env = append(c.Env, v1.EnvVar{
				Name:  "DUBBO_CA_CERT_PATH",
				Value: "/var/run/secrets/dubbo-ca-cert/ca.crt",
			})
			c.Env = append(c.Env, v1.EnvVar{
				Name:  "DUBBO_OIDC_TOKEN",
				Value: "/var/run/secrets/dubbo-ca-token/token",
			})

			c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
				Name:      "dubbo-ca-token",
				MountPath: "/var/run/secrets/dubbo-ca-token",
			})
			c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
				Name:      "dubbo-ca-cert",
				MountPath: "/var/run/secrets/dubbo-ca-cert",
			})

			targetContainers = append(targetContainers, c)
		}
		target.Spec.Containers = targetContainers
	}

	return target, nil
}
