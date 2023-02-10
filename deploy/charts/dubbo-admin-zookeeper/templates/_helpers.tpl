{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "dubbo-admin.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "dubbo-admin.fullname" -}}
{{- $name := default .Chart.Name }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "dubbo-admin.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts
*/}}
{{- define "dubbo-admin.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}
{{/*
Common labels
*/}}


{{- define "dubbo-admin.labels" -}}
helm.sh/chart: {{ include "dubbo-admin.chart" . }}
{{ include "dubbo-admin.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


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


{{- define "dubbo-admin.selectorLabels" -}}
app.kubernetes.io/name: {{ include "dubbo-admin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{/*
Create the name of the service account to use
*/}}
{{- define "dubbo-admin.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "dubbo-admin.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{- define "dubbo-admin.serviceAccountNameTest" -}}
{{- if .Values.serviceAccount.create }}
{{- default (print (include "dubbo-admin.fullname" .) "-test") .Values.serviceAccount.nameTest }}
{{- else }}
{{- default "default" .Values.serviceAccount.nameTest }}
{{- end }}
{{- end }}


{{/*
Formats imagePullSecrets. Input is (dict "root" . "imagePullSecrets" .{specific imagePullSecrets})
*/}}
{{- define "dubbo-admin.imagePullSecrets" -}}
{{- $root := .root }}
{{- range (concat .root.Values.global.imagePullSecrets .imagePullSecrets) }}
{{- if eq (typeOf .) "map[string]interface {}" }}
- {{ toYaml (dict "name" (tpl .name $root)) | trim }}
{{- else }}
- name: {{ tpl . $root }}
{{- end }}
{{- end }}
{{- end }}


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


{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "zookeeper.name" -}}
{{- default .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "zookeeper.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}


{{/*
Return ZooKeeper Namespace to use
*/}}
{{- define "zookeeper.namespace" -}}
{{- if .Values.namespaceOverride -}}
    {{- .Values.namespaceOverride -}}
{{- else -}}
    {{- .Release.Namespace -}}
{{- end -}}
{{- end -}}


{{/*
Return the appropriate apiVersion for statefulset.
*/}}
{{- define "zookeeper.statefulset.apiVersion" -}}
{{- if semverCompare "<1.14-0" (include "zookeeper.kubeVersion" .) -}}
{{- print "apps/v1beta1" -}}
{{- else -}}
{{- print "apps/v1" -}}
{{- end -}}
{{- end -}}


{{/*
Labels to use on deploy.spec.selector.matchLabels and svc.spec.selector
*/}}
{{- define "zookeeper.matchLabels" -}}
app.kubernetes.io/name: {{ include "zookeeper.name" . }}
{{- end -}}

{{- define "zookeeper.labels" -}}
app.kubernetes.io/name: {{ include "zookeeper.name" . }}
helm.sh/chart: {{ include "zookeeper.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "zookeeper.chart" -}}
{{- end -}}
{{/*
Return the target Kubernetes version
*/}}
{{- define "zookeeper.kubeVersion" -}}
{{- if .Values.global }}
    {{- if .Values.global.kubeVersion }}
    {{- .Values.global.kubeVersion -}}
    {{- else }}
    {{- default .Capabilities.KubeVersion.Version .Values.kubeVersion -}}
    {{- end -}}
{{- else }}
{{- default .Capabilities.KubeVersion.Version .Values.kubeVersion -}}
{{- end -}}
{{- end -}}


{{/* vim: set filetype=mustache: */}}
{{/*
Renders a value that contains template.
Usage:
{{ include "zookeeper.tplvalues" ( dict "value" .Values.path.to.the.Value "context" $) }}
*/}}
{{- define "zookeeper.tplvalues" -}}
    {{- if typeIs "string" .value }}
        {{- tpl .value .context }}
    {{- else }}
        {{- tpl (.value | toYaml) .context }}
    {{- end }}
{{- end -}}


{{/*
Return the ZooKeeper configuration ConfigMap name
*/}}
{{- define "zookeeper.configmapName" -}}
{{- if .Values.existingConfigmap -}}
    {{- printf "%s" (tpl .Values.existingConfigmap $) -}}
{{- else -}}
    {{- printf "%s" (include "zookeeper.fullname" .) -}}
{{- end -}}
{{- end -}}


{{/* vim: set filetype=mustache: */}}
{{/*
Return  the proper Storage Class
{{ include "common.storage.class" ( dict "persistence" .Values.path.to.the.persistence "global" $) }}
*/}}
{{- define "zookeeper.storage.class" -}}

{{- $storageClass := .vpersistence.storageClass -}}
{{- if .global -}}
    {{- if .global.storageClass -}}
        {{- $storageClass = .global.storageClass -}}
    {{- end -}}
{{- end -}}

{{- if $storageClass -}}
  {{- if (eq "-" $storageClass) -}}
      {{- printf "storageClassName: \"\"" -}}
  {{- else }}
      {{- printf "storageClassName: %s" $storageClass -}}
  {{- end -}}
{{- end -}}

{{- end -}}
