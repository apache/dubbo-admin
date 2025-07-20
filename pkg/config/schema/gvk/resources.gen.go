package gvk

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/apache/dubbo-admin/operator/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/schema/gvr"
)

var (
	Namespace                      = config.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}
	CustomResourceDefinition       = config.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
	MutatingWebhookConfiguration   = config.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"}
	ValidatingWebhookConfiguration = config.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}
	Deployment                     = config.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	StatefulSet                    = config.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	DaemonSet                      = config.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}
	Job                            = config.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}
	ConfigMap                      = config.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	Secret                         = config.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
	Service                        = config.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}
	ServiceAccount                 = config.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}
)

func ToGVR(g config.GroupVersionKind) (schema.GroupVersionResource, bool) {
	switch g {
	case CustomResourceDefinition:
		return gvr.CustomResourceDefinition, true
	case MutatingWebhookConfiguration:
		return gvr.MutatingWebhookConfiguration, true
	case ValidatingWebhookConfiguration:
		return gvr.ValidatingWebhookConfiguration, true
	case Deployment:
		return gvr.Deployment, true
	case StatefulSet:
		return gvr.StatefulSet, true
	case DaemonSet:
		return gvr.DaemonSet, true
	case ConfigMap:
		return gvr.ConfigMap, true
	case Secret:
		return gvr.Secret, true
	case Service:
		return gvr.Service, true
	case ServiceAccount:
		return gvr.ServiceAccount, true
	case Namespace:
		return gvr.Namespace, true
	case Job:
		return gvr.Job, true
	}
	return schema.GroupVersionResource{}, false
}
func MustToGVR(g config.GroupVersionKind) schema.GroupVersionResource {
	r, ok := ToGVR(g)
	if !ok {
		panic("unknown kind: " + g.String())
	}
	return r
}

func FromGVR(g schema.GroupVersionResource) (config.GroupVersionKind, bool) {
	switch g {
	case gvr.CustomResourceDefinition:
		return CustomResourceDefinition, true
	case gvr.Namespace:
		return Namespace, true
	case gvr.Deployment:
		return Deployment, true
	case gvr.StatefulSet:
		return StatefulSet, true
	case gvr.DaemonSet:
		return DaemonSet, true
	case gvr.Job:
		return Job, true
	}
	return config.GroupVersionKind{}, false
}

func MustFromGVR(g schema.GroupVersionResource) config.GroupVersionKind {
	r, ok := FromGVR(g)
	if !ok {
		panic("unknown kind: " + g.String())
	}
	return r
}
