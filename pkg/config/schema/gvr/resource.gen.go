package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	CustomResourceDefinition       = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	MutatingWebhookConfiguration   = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "MutatingWebhookConfiguration"}
	ValidatingWebhookConfiguration = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "ValidatingWebhookConfiguration"}
	Deployment                     = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	StatefulSet                    = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	DaemonSet                      = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
	Job                            = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	Namespace                      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	ConfigMap                      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	Secret                         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	Service                        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	ServiceAccount                 = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
)
