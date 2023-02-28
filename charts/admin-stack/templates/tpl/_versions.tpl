{{/*
Return the appropriate apiVersion for rbac.
*/}}
{{- define "dubbo-admin.rbac.apiVersion" -}}
{{- if $.Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1" }}
{{- print "rbac.authorization.k8s.io/v1" }}
{{- else }}
{{- print "rbac.authorization.k8s.io/v1beta1" }}
{{- end }}
{{- end }}


{{/*
Return the appropriate apiVersion for ingress.
*/}}
{{- define "dubbo-admin.ingress.apiVersion" -}}
{{- if and ($.Capabilities.APIVersions.Has "networking.k8s.io/v1") (semverCompare ">= 1.19-0" .Capabilities.KubeVersion.Version) }}
{{- print "networking.k8s.io/v1" }}
{{- else if $.Capabilities.APIVersions.Has "networking.k8s.io/v1beta1" }}
{{- print "networking.k8s.io/v1beta1" }}
{{- else }}
{{- print "extensions/v1beta1" }}
{{- end }}
{{- end }}


{{/*
Return the appropriate apiVersion for podDisruptionBudget.
*/}}
{{- define "dubbo-admin.podDisruptionBudget.apiVersion" -}}
{{- if $.Capabilities.APIVersions.Has "policy/v1/PodDisruptionBudget" }}
{{- print "policy/v1" }}
{{- else }}
{{- print "policy/v1beta1" }}
{{- end }}
{{- end }}


{{- define "zookeeper.statefulset.apiVersion" -}}
{{- if semverCompare "<1.14-0" (include "zookeeper.kubeVersion" .) -}}
{{- print "apps/v1beta1" -}}
{{- else -}}
{{- print "apps/v1" -}}
{{- end -}}
{{- end -}}


{{/*
Return the appropriate apiVersion for networkpolicy.
*/}}
{{- define "zookeeper.networkPolicy.apiVersion" -}}
{{- if semverCompare "<1.7-0" (include "zookeeper.kubeVersion" .) -}}
{{- print "extensions/v1beta1" -}}
{{- else -}}
{{- print "networking.k8s.io/v1" -}}
{{- end -}}
{{- end -}}


{{/*
Return the appropriate apiVersion for networkpolicy.
*/}}
{{- define "nacos.networkPolicy.apiVersion" -}}
{{- if semverCompare "<1.7-0" (include "nacos.kubeVersion" .) -}}
{{- print "extensions/v1beta1" -}}
{{- else -}}
{{- print "networking.k8s.io/v1" -}}
{{- end -}}
{{- end -}}


{{/*
Return the appropriate apiVersion for poddisruptionbudget.
*/}}
{{- define "zookeeper.policy.apiVersion" -}}
{{- if semverCompare "<1.21-0" (include "zookeeper.kubeVersion" .) -}}
{{- print "policy/v1beta1" -}}
{{- else -}}
{{- print "policy/v1" -}}
{{- end -}}
{{- end -}}


{{/*
Return the appropriate apiVersion for poddisruptionbudget.
*/}}
{{- define "nacos.policy.apiVersion" -}}
{{- if semverCompare "<1.21-0" (include "nacos.kubeVersion" .) -}}
{{- print "policy/v1beta1" -}}
{{- else -}}
{{- print "policy/v1" -}}
{{- end -}}
{{- end -}}



