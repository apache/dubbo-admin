{{/*
Return if ingress is stable.
*/}}
{{- define "dubbo-admin.ingress.isStable" -}}
{{- eq (include "dubbo-admin.ingress.apiVersion" .) "networking.k8s.io/v1" }}
{{- end }}


{{/*
Return if ingress supports ingressClassName.
*/}}
{{- define "dubbo-admin.ingress.supportsIngressClassName" -}}
{{- or (eq (include "dubbo-admin.ingress.isStable" .) "true") (and (eq (include "dubbo-admin.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" .Capabilities.KubeVersion.Version)) }}
{{- end }}


{{/*
Return if ingress supports pathType.
*/}}
{{- define "dubbo-admin.ingress.supportsPathType" -}}
{{- or (eq (include "dubbo-admin.ingress.isStable" .) "true") (and (eq (include "dubbo-admin.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" .Capabilities.KubeVersion.Version)) }}
{{- end }}