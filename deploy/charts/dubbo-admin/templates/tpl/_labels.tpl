{{- define "dubbo-admin.labels" -}}
helm.sh/chart: {{ include "dubbo-admin.chart" . }}
{{ include "dubbo-admin.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{- define "zookeeper.labels" -}}
app.kubernetes.io/name: {{ include "zookeeper.name" . }}
helm.sh/chart: {{ include "zookeeper.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}


{{- define "nacos.labels" -}}
app.kubernetes.io/name: {{ include "nacos.name" . }}
helm.sh/chart: {{ include "nacos.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}