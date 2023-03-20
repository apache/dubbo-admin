{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "zipkin.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "zipkin.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "zipkin.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
Common labels
*/}}
{{- define "zipkin.common.labels" -}}
helm.sh/chart: {{ include "zipkin.chart" . }}
{{ include "zipkin.common.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}


{{/*
Common Selector labels
*/}}
{{- define "zipkin.common.selectorLabels" -}}
app.kubernetes.io/name: {{ include "zipkin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "zipkin.collector.serviceAccountName" -}}
{{- if .Values.collector.serviceAccount.create -}}
    {{ default (include "zipkin.collector.fullname" .) .Values.collector.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.collector.serviceAccount.name }}
{{- end -}}
{{- end -}}


{{/*
Collector name
*/}}
{{- define "zipkin.collector.fullname" -}}
{{- printf "%s-%s" (include "zipkin.fullname" .) "collector" -}}
{{- end -}}

{{/*
Collector service address
*/}}
{{- define "zipkin.collector.service.uri" -}}
{{- printf "http://%s:%d" (include "zipkin.collector.fullname" .) (.Values.collector.service.port | int ) -}}
{{- end -}}

{{/*
Collector labels
*/}}
{{- define "zipkin.collector.labels" -}}
app.kubernetes.io/component: collector
app.kubernetes.io/version: {{ .Values.collector.image.tag | default .Chart.AppVersion | quote }}
{{ include "zipkin.common.labels" . }}
{{- end -}}


{{/*
Collector Selector labels
*/}}
{{- define "zipkin.collector.selectorLabels" -}}
app.kubernetes.io/component: collector
{{ include "zipkin.common.selectorLabels" . }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "zipkin.dependencies.serviceAccountName" -}}
{{- if .Values.dependencies.serviceAccount.create -}}
    {{ default (include "zipkin.dependencies.fullname" .) .Values.dependencies.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.dependencies.serviceAccount.name }}
{{- end -}}
{{- end -}}


{{/*
dependencies name
*/}}
{{- define "zipkin.dependencies.fullname" -}}
{{- printf "%s-%s" (include "zipkin.fullname" .) "dependencies" -}}
{{- end -}}


{{/*
dependencies labels
*/}}
{{- define "zipkin.dependencies.labels" -}}
app.kubernetes.io/component: dependencies
app.kubernetes.io/version: {{ .Values.dependencies.image.tag | quote }}
{{ include "zipkin.common.labels" . }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "zipkin.ui.serviceAccountName" -}}
{{- if .Values.ui.serviceAccount.create -}}
    {{ default (include "zipkin.ui.fullname" .) .Values.ui.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.ui.serviceAccount.name }}
{{- end -}}
{{- end -}}


{{/*
ui name
*/}}
{{- define "zipkin.ui.fullname" -}}
{{- printf "%s-%s" (include "zipkin.fullname" .) "ui" -}}
{{- end -}}


{{/*
ui labels
*/}}
{{- define "zipkin.ui.labels" -}}
app.kubernetes.io/component: ui
app.kubernetes.io/version: {{ .Values.ui.image.tag | default .Chart.AppVersion | quote }}
{{ include "zipkin.common.labels" . }}
{{- end -}}


{{/*
ui Selector labels
*/}}
{{- define "zipkin.ui.selectorLabels" -}}
app.kubernetes.io/component: ui
{{ include "zipkin.common.selectorLabels" . }}
{{- end -}}
