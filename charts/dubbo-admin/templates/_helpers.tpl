{{/*
*/}}
{{- define "dubbo-admin.name" -}}
{{- if .Values.nameOverride }}
{{- else }}
{{- printf "dubbo-admin" -}}
{{- end -}}
{{- end -}}

{{- define "dubbo-admin.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- else }}
{{- printf "default" }}
{{- end -}}
{{- end -}}

{{/*
*/}}
{{- define "dubbo-admin.labels" -}}
app.kubernetes.io/name: {{ template "dubbo-admin.name" . }}
helm.sh/chart: {{ include "dubbo-admin.name" . }}-{{ .Values.image.tag }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
*/}}
{{- define "dubbo-admin.matchLabels" -}}
app.kubernetes.io/name: {{ template "dubbo-admin.name" . }}
helm.sh/chart: {{ include "dubbo-admin.name" . }}-{{ .Values.image.tag }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "dubbo-admin.podDisruptionBudget.apiVersion" -}}
{{- if $.Capabilities.APIVersions.Has "policy/v1/PodDisruptionBudget" }}
{{- print "policy/v1" }}
{{- else }}
{{- print "policy/v1beta1" }}
{{- end }}
{{- end }}