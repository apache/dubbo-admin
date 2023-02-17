{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "dubbo-admin.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "zookeeper.fullname" -}}
{{- if .Values.zookeeper.zoo_name -}}
{{- .Values.zookeeper.zoo_name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Values.zookeeper.zoo_name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}


{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nacos.fullname" -}}
{{- if .Values.nacos.name -}}
{{- .Values.nacos.name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Values.nacos.name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}


{{/*
Labels to use on sts.spec.selector.matchLabels and svc.spec.selector
*/}}
{{- define "zookeeper.matchLabels" -}}
app.kubernetes.io/name: {{ include "zookeeper.name" . }}
{{- end -}}

{{/*
Labels to use on sts.spec.selector.matchLabels and svc.spec.selector
*/}}
{{- define "nacos.matchLabels" -}}
app.kubernetes.io/name: {{ include "nacos.name" . }}
{{- end -}}


{{- define "dubbo-admin.selectorLabels" -}}
app.kubernetes.io/name: {{ include "dubbo-admin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{/*
Create the name of the service account to use
*/}}
{{- define "dubbo-admin.serviceAccountName" -}}
{{- if .Values.serviceAccount.enabled }}
{{- default (include "dubbo-admin.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{- define "dubbo-admin.serviceAccountNameTest" -}}
{{- if .Values.serviceAccount.enabled }}
{{- default (print (include "dubbo-admin.fullname" .) "-test") .Values.serviceAccount.nameTest }}
{{- else }}
{{- default "default" .Values.serviceAccount.nameTest }}
{{- end }}
{{- end }}


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