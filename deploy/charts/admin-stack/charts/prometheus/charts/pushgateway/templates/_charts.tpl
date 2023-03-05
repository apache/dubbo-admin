{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "prometheus-pushgateway.fullname" -}}
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

{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "prometheus-pushgateway.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "prometheus-pushgateway.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create default labels
*/}}
{{- define "prometheus-pushgateway.defaultLabels" -}}
helm.sh/chart: {{ include "prometheus-pushgateway.chart" . }}
{{ include "prometheus-pushgateway.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.podLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "prometheus.pushgateway.matchLabels" -}}
component: {{ .Values.pushgateway.name | quote }}
{{ include "prometheus.common.matchLabels" . }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "prometheus-pushgateway.selectorLabels" -}}
app.kubernetes.io/name: {{ include "prometheus-pushgateway.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "prometheus-pushgateway.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "prometheus-pushgateway.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Returns pod spec
*/}}
{{- define "prometheus-pushgateway.podSpec" -}}
serviceAccountName: {{ include "prometheus-pushgateway.serviceAccountName" . }}
{{- with .Values.priorityClassName }}
priorityClassName: {{ . | quote }}
{{- end }}
{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.extraInitContainers }}
initContainers:
  {{- toYaml . | nindent 2 }}
{{- end }}
containers:
  {{- with .Values.extraContainers }}
  {{- toYaml . | nindent 2 }}
  {{- end }}
  - name: pushgateway
    image: "{{ .Values.pushgateway.image.repository }}:{{ .Values.pushgateway.image.tag | default .Chart.AppVersion }}"
    imagePullPolicy: {{ .Values.pushgateway.image.pullPolicy }}
    {{- with .Values.extraVars }}
    env:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.extraArgs }}
    args:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    ports:
      - name: metrics
        containerPort: 9091
        protocol: TCP
    {{- if .Values.liveness.enabled }}
    livenessProbe:
      {{- toYaml .Values.liveness.probe | nindent 6 }}
    {{- end }}
    {{- if .Values.readiness.enabled }}
    readinessProbe:
      {{- toYaml .Values.readiness.probe | nindent 6 }}
    {{- end }}
    {{- with .Values.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.containerSecurityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: storage-volume
        mountPath: "{{ .Values.pushgateway.persistentVolume.mountPath }}"
        subPath: "{{ .Values.pushgateway.persistentVolume.subPath }}"
      {{- with .Values.pushgateway.extraVolumeMounts }}
      {{- toYaml . | nindent 6 }}
      {{- end }}
{{- with .Values.nodeSelector }}
nodeSelector:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.tolerations }}
tolerations:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.affinity }}
affinity:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.topologySpreadConstraints }}
topologySpreadConstraints:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.securityContext }}
securityContext:
  {{- toYaml . | nindent 2 }}
{{- end }}
volumes:
  {{- $storageVolumeAsPVCTemplate := and .Values.runAsStatefulSet .Values.pushgateway.persistentVolume.enabled -}}
  {{- if not $storageVolumeAsPVCTemplate }}
  - name: storage-volume
  {{- if .Values.pushgateway.persistentVolume.enabled }}
    persistentVolumeClaim:
      claimName: {{ if .Values.pushgateway.persistentVolume.existingClaim }}{{ .Values.pushgateway.persistentVolume.existingClaim }}{{- else }}{{ include "prometheus-pushgateway.fullname" . }}{{- end }}
  {{- else }}
    emptyDir: {}
  {{- end }}
  {{- end }}
  {{- if .Values.pushgateway.extraVolumes }}
  {{- toYaml .Values.pushgateway.extraVolumes  | nindent 2 }}
  {{- else if $storageVolumeAsPVCTemplate }}
  []
  {{- end }}
{{- end }}
