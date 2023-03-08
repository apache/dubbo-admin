{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "grafana.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "grafana.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "grafana.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the service account
*/}}
{{- define "grafana.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "grafana.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "grafana.serviceAccountNameTest" -}}
{{- if .Values.serviceAccount.create }}
{{- default (print (include "grafana.fullname" .) "-test") .Values.serviceAccount.nameTest }}
{{- else }}
{{- default "default" .Values.serviceAccount.nameTest }}
{{- end }}
{{- end }}

{{/*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts
*/}}
{{- define "grafana.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "grafana.labels" -}}
helm.sh/chart: {{ include "grafana.chart" . }}
{{ include "grafana.selectorLabels" . }}
{{- if or .Chart.AppVersion .Values.image.tag }}
app.kubernetes.io/version: {{ mustRegexReplaceAllLiteral "@sha.*" .Values.image.tag "" | default .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.extraLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "grafana.selectorLabels" -}}
app.kubernetes.io/name: {{ include "grafana.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "grafana.imageRenderer.labels" -}}
helm.sh/chart: {{ include "grafana.chart" . }}
{{ include "grafana.imageRenderer.selectorLabels" . }}
{{- if or .Chart.AppVersion .Values.image.tag }}
app.kubernetes.io/version: {{ mustRegexReplaceAllLiteral "@sha.*" .Values.image.tag "" | default .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{/*
Selector labels ImageRenderer
*/}}
{{- define "grafana.imageRenderer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "grafana.name" . }}-image-renderer
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{/*
Looks if there's an existing secret and reuse its password. If not it generates
new password and use it.
*/}}
{{- define "grafana.password" -}}
{{- $secret := (lookup "v1" "Secret" (include "grafana.namespace" .) (include "grafana.fullname" .) ) }}
{{- if $secret }}
{{- index $secret "data" "admin-password" }}
{{- else }}
{{- (randAlphaNum 40) | b64enc | quote }}
{{- end }}
{{- end }}


{{/*
Return the appropriate apiVersion for rbac.
*/}}
{{- define "grafana.rbac.apiVersion" -}}
{{- if $.Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1" }}
{{- print "rbac.authorization.k8s.io/v1" }}
{{- else }}
{{- print "rbac.authorization.k8s.io/v1beta1" }}
{{- end }}
{{- end }}


{{/*
Return the appropriate apiVersion for ingress.
*/}}
{{- define "grafana.ingress.apiVersion" -}}
{{- if and ($.Capabilities.APIVersions.Has "networking.k8s.io/v1") (semverCompare ">= 1.19-0" .Capabilities.KubeVersion.Version) }}
{{- print "networking.k8s.io/v1" }}
{{- else if $.Capabilities.APIVersions.Has "networking.k8s.io/v1beta1" }}
{{- print "networking.k8s.io/v1beta1" }}
{{- else }}
{{- print "extensions/v1beta1" }}
{{- end }}
{{- end }}


{{/*
Return the appropriate apiVersion for Horizontal Pod Autoscaler.
*/}}
{{- define "grafana.hpa.apiVersion" -}}
{{- if $.Capabilities.APIVersions.Has "autoscaling/v2/HorizontalPodAutoscaler" }}
{{- print "autoscaling/v2" }}
{{- else if $.Capabilities.APIVersions.Has "autoscaling/v2beta2/HorizontalPodAutoscaler" }}
{{- print "autoscaling/v2beta2" }}
{{- else }}
{{- print "autoscaling/v2beta1" }}
{{- end }}
{{- end }}


{{/*
Return the appropriate apiVersion for podDisruptionBudget.
*/}}
{{- define "grafana.podDisruptionBudget.apiVersion" -}}
{{- if $.Capabilities.APIVersions.Has "policy/v1/PodDisruptionBudget" }}
{{- print "policy/v1" }}
{{- else }}
{{- print "policy/v1beta1" }}
{{- end }}
{{- end }}


{{/*
Return if ingress is stable.
*/}}
{{- define "grafana.ingress.isStable" -}}
{{- eq (include "grafana.ingress.apiVersion" .) "networking.k8s.io/v1" }}
{{- end }}


{{/*
Return if ingress supports ingressClassName.
*/}}
{{- define "grafana.ingress.supportsIngressClassName" -}}
{{- or (eq (include "grafana.ingress.isStable" .) "true") (and (eq (include "grafana.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" .Capabilities.KubeVersion.Version)) }}
{{- end }}


{{/*
Return if ingress supports pathType.
*/}}
{{- define "grafana.ingress.supportsPathType" -}}
{{- or (eq (include "grafana.ingress.isStable" .) "true") (and (eq (include "grafana.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" .Capabilities.KubeVersion.Version)) }}
{{- end }}


{{/*
Formats imagePullSecrets. Input is (dict "root" . "imagePullSecrets" .{specific imagePullSecrets})
*/}}
{{- define "grafana.imagePullSecrets" -}}
{{- $root := .root }}
{{- range (concat .root.Values.global.imagePullSecrets .imagePullSecrets) }}
{{- if eq (typeOf .) "map[string]interface {}" }}
- {{ toYaml (dict "name" (tpl .name $root)) | trim }}
{{- else }}
- name: {{ tpl . $root }}
{{- end }}
{{- end }}
{{- end }}

{{- define "grafana.pod" -}}
{{- $sts := list "sts" "StatefulSet" "statefulset" -}}
{{- $root := . -}}
{{- with .Values.schedulerName }}
schedulerName: "{{ . }}"
{{- end }}
serviceAccountName: {{ include "grafana.serviceAccountName" . }}
automountServiceAccountToken: {{ .Values.serviceAccount.autoMount }}
{{- with .Values.securityContext }}
securityContext:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.hostAliases }}
hostAliases:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.priorityClassName }}
priorityClassName: {{ . }}
{{- end }}
{{- if ( or .Values.persistence.enabled .Values.dashboards .Values.extraInitContainers (and .Values.sidecar.datasources.enabled .Values.sidecar.datasources.initDatasources) (and .Values.sidecar.notifiers.enabled .Values.sidecar.notifiers.initNotifiers)) }}
initContainers:
{{- end }}
{{- if ( and .Values.persistence.enabled .Values.initChownData.enabled ) }}
  - name: init-chown-data
    {{- if .Values.initChownData.image.sha }}
    image: "{{ .Values.initChownData.image.repository }}:{{ .Values.initChownData.image.tag }}@sha256:{{ .Values.initChownData.image.sha }}"
    {{- else }}
    image: "{{ .Values.initChownData.image.repository }}:{{ .Values.initChownData.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.initChownData.image.pullPolicy }}
    {{- with .Values.initChownData.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    command:
      - chown
      - -R
      - {{ .Values.securityContext.runAsUser }}:{{ .Values.securityContext.runAsGroup }}
      - /var/lib/grafana
    {{- with .Values.initChownData.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: storage
        mountPath: "/var/lib/grafana"
        {{- with .Values.persistence.subPath }}
        subPath: {{ tpl . $root }}
        {{- end }}
{{- end }}
{{- if .Values.dashboards }}
  - name: download-dashboards
    {{- if .Values.downloadDashboardsImage.sha }}
    image: "{{ .Values.downloadDashboardsImage.repository }}:{{ .Values.downloadDashboardsImage.tag }}@sha256:{{ .Values.downloadDashboardsImage.sha }}"
    {{- else }}
    image: "{{ .Values.downloadDashboardsImage.repository }}:{{ .Values.downloadDashboardsImage.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.downloadDashboardsImage.pullPolicy }}
    command: ["/bin/sh"]
    args: [ "-c", "mkdir -p /var/lib/grafana/dashboards/default && /bin/sh -x /etc/grafana/download_dashboards.sh" ]
    {{- with .Values.downloadDashboards.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    env:
      {{- range $key, $value := .Values.downloadDashboards.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- range $key, $value := .Values.downloadDashboards.envValueFrom }}
      - name: {{ $key | quote }}
        valueFrom:
          {{- tpl (toYaml $value) $ | nindent 10 }}
      {{- end }}
    {{- with .Values.downloadDashboards.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.downloadDashboards.envFromSecret }}
    envFrom:
      - secretRef:
          name: {{ tpl . $root }}
    {{- end }}
    volumeMounts:
      - name: config
        mountPath: "/etc/grafana/download_dashboards.sh"
        subPath: download_dashboards.sh
      - name: storage
        mountPath: "/var/lib/grafana"
        {{- with .Values.persistence.subPath }}
        subPath: {{ tpl . $root }}
        {{- end }}
      {{- range .Values.extraSecretMounts }}
      - name: {{ .name }}
        mountPath: {{ .mountPath }}
        readOnly: {{ .readOnly }}
      {{- end }}
{{- end }}
{{- if and .Values.sidecar.datasources.enabled .Values.sidecar.datasources.initDatasources }}
  - name: {{ include "grafana.name" . }}-init-sc-datasources
    {{- if .Values.sidecar.image.sha }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}@sha256:{{ .Values.sidecar.image.sha }}"
    {{- else }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.sidecar.imagePullPolicy }}
    env:
      {{- range $key, $value := .Values.sidecar.datasources.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- if .Values.sidecar.datasources.ignoreAlreadyProcessed }}
      - name: IGNORE_ALREADY_PROCESSED
        value: "true"
      {{- end }}
      - name: METHOD
        value: "LIST"
      - name: LABEL
        value: "{{ .Values.sidecar.datasources.label }}"
      {{- with .Values.sidecar.datasources.labelValue }}
      - name: LABEL_VALUE
        value: {{ quote . }}
      {{- end }}
      {{- if or .Values.sidecar.logLevel .Values.sidecar.datasources.logLevel }}
      - name: LOG_LEVEL
        value: {{ default .Values.sidecar.logLevel .Values.sidecar.datasources.logLevel }}
      {{- end }}
      - name: FOLDER
        value: "/etc/grafana/provisioning/datasources"
      - name: RESOURCE
        value: {{ quote .Values.sidecar.datasources.resource }}
      {{- with .Values.sidecar.enableUniqueFilenames }}
      - name: UNIQUE_FILENAMES
        value: "{{ . }}"
      {{- end }}
      {{- if .Values.sidecar.datasources.searchNamespace }}
      - name: NAMESPACE
        value: "{{ tpl (.Values.sidecar.datasources.searchNamespace | join ",") . }}"
      {{- end }}
      {{- with .Values.sidecar.skipTlsVerify }}
      - name: SKIP_TLS_VERIFY
        value: "{{ . }}"
      {{- end }}
    {{- with .Values.sidecar.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: sc-datasources-volume
        mountPath: "/etc/grafana/provisioning/datasources"
{{- end }}
{{- if and .Values.sidecar.notifiers.enabled .Values.sidecar.notifiers.initNotifiers }}
  - name: {{ include "grafana.name" . }}-init-sc-notifiers
    {{- if .Values.sidecar.image.sha }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}@sha256:{{ .Values.sidecar.image.sha }}"
    {{- else }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.sidecar.imagePullPolicy }}
    env:
      {{- range $key, $value := .Values.sidecar.notifiers.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- if .Values.sidecar.notifiers.ignoreAlreadyProcessed }}
      - name: IGNORE_ALREADY_PROCESSED
        value: "true"
      {{- end }}
      - name: METHOD
        value: LIST
      - name: LABEL
        value: "{{ .Values.sidecar.notifiers.label }}"
      {{- with .Values.sidecar.notifiers.labelValue }}
      - name: LABEL_VALUE
        value: {{ quote . }}
      {{- end }}
      {{- if or .Values.sidecar.logLevel .Values.sidecar.notifiers.logLevel }}
      - name: LOG_LEVEL
        value: {{ default .Values.sidecar.logLevel .Values.sidecar.notifiers.logLevel }}
      {{- end }}
      - name: FOLDER
        value: "/etc/grafana/provisioning/notifiers"
      - name: RESOURCE
        value: {{ quote .Values.sidecar.notifiers.resource }}
      {{- with .Values.sidecar.enableUniqueFilenames }}
      - name: UNIQUE_FILENAMES
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.notifiers.searchNamespace }}
      - name: NAMESPACE
        value: "{{ tpl (. | join ",") $root }}"
      {{- end }}
      {{- with .Values.sidecar.skipTlsVerify }}
      - name: SKIP_TLS_VERIFY
        value: "{{ . }}"
      {{- end }}
    {{- with .Values.sidecar.livenessProbe }}
    livenessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.readinessProbe }}
    readinessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: sc-notifiers-volume
        mountPath: "/etc/grafana/provisioning/notifiers"
{{- end}}
{{- with .Values.extraInitContainers }}
  {{- tpl (toYaml .) $root | nindent 2 }}
{{- end }}
{{- if or .Values.image.pullSecrets .Values.global.imagePullSecrets }}
imagePullSecrets:
  {{- include "grafana.imagePullSecrets" (dict "root" $root "imagePullSecrets" .Values.image.pullSecrets) | nindent 2 }}
{{- end }}
{{- if not .Values.enableKubeBackwardCompatibility }}
enableServiceLinks: {{ .Values.enableServiceLinks }}
{{- end }}
containers:
{{- if .Values.sidecar.alerts.enabled }}
  - name: {{ include "grafana.name" . }}-sc-alerts
    {{- if .Values.sidecar.image.sha }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}@sha256:{{ .Values.sidecar.image.sha }}"
    {{- else }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.sidecar.imagePullPolicy }}
    env:
      {{- range $key, $value := .Values.sidecar.alerts.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- if .Values.sidecar.alerts.ignoreAlreadyProcessed }}
      - name: IGNORE_ALREADY_PROCESSED
        value: "true"
      {{- end }}
      - name: METHOD
        value: {{ .Values.sidecar.alerts.watchMethod }}
      - name: LABEL
        value: "{{ .Values.sidecar.alerts.label }}"
      {{- with .Values.sidecar.alerts.labelValue }}
      - name: LABEL_VALUE
        value: {{ quote . }}
      {{- end }}
      {{- if or .Values.sidecar.logLevel .Values.sidecar.alerts.logLevel }}
      - name: LOG_LEVEL
        value: {{ default .Values.sidecar.logLevel .Values.sidecar.alerts.logLevel }}
      {{- end }}
      - name: FOLDER
        value: "/etc/grafana/provisioning/alerting"
      - name: RESOURCE
        value: {{ quote .Values.sidecar.alerts.resource }}
      {{- with .Values.sidecar.enableUniqueFilenames }}
      - name: UNIQUE_FILENAMES
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.alerts.searchNamespace }}
      - name: NAMESPACE
        value: {{ . | join "," | quote }}
      {{- end }}
      {{- with .Values.sidecar.alerts.skipTlsVerify }}
      - name: SKIP_TLS_VERIFY
        value: {{ quote . }}
      {{- end }}
      {{- with .Values.sidecar.alerts.script }}
      - name: SCRIPT
        value: {{ quote . }}
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_USER) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_USERNAME
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.userKey | default "admin-user" }}
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_PASSWORD) (not .Values.env.GF_SECURITY_ADMIN_PASSWORD__FILE) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_PASSWORD
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.passwordKey | default "admin-password" }}
      {{- end }}
      {{- if not .Values.sidecar.alerts.skipReload }}
      - name: REQ_URL
        value: {{ .Values.sidecar.alerts.reloadURL }}
      - name: REQ_METHOD
        value: POST
      {{- end }}
      {{- if .Values.sidecar.alerts.watchServerTimeout }}
      {{- if ne .Values.sidecar.alerts.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.alerts.watchServerTimeout with .Values.sidecar.alerts.watchMethod %s" .Values.sidecar.alerts.watchMethod) }}
      {{- end }}
      - name: WATCH_SERVER_TIMEOUT
        value: "{{ .Values.sidecar.alerts.watchServerTimeout }}"
      {{- end }}
      {{- if .Values.sidecar.alerts.watchClientTimeout }}
      {{- if ne .Values.sidecar.alerts.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.alerts.watchClientTimeout with .Values.sidecar.alerts.watchMethod %s" .Values.sidecar.alerts.watchMethod) }}
      {{- end }}
      - name: WATCH_CLIENT_TIMEOUT
        value: "{{ .Values.sidecar.alerts.watchClientTimeout }}"
      {{- end }}
    {{- with .Values.sidecar.livenessProbe }}
    livenessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.readinessProbe }}
    readinessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: sc-alerts-volume
        mountPath: "/etc/grafana/provisioning/alerting"
{{- end}}
{{- if .Values.sidecar.dashboards.enabled }}
  - name: {{ include "grafana.name" . }}-sc-dashboard
    {{- if .Values.sidecar.image.sha }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}@sha256:{{ .Values.sidecar.image.sha }}"
    {{- else }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.sidecar.imagePullPolicy }}
    env:
      {{- range $key, $value := .Values.sidecar.dashboards.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- if .Values.sidecar.dashboards.ignoreAlreadyProcessed }}
      - name: IGNORE_ALREADY_PROCESSED
        value: "true"
      {{- end }}
      - name: METHOD
        value: {{ .Values.sidecar.dashboards.watchMethod }}
      - name: LABEL
        value: "{{ .Values.sidecar.dashboards.label }}"
      {{- with .Values.sidecar.dashboards.labelValue }}
      - name: LABEL_VALUE
        value: {{ quote . }}
      {{- end }}
      {{- if or .Values.sidecar.logLevel .Values.sidecar.dashboards.logLevel }}
      - name: LOG_LEVEL
        value: {{ default .Values.sidecar.logLevel .Values.sidecar.dashboards.logLevel }}
      {{- end }}
      - name: FOLDER
        value: "{{ .Values.sidecar.dashboards.folder }}{{- with .Values.sidecar.dashboards.defaultFolderName }}/{{ . }}{{- end }}"
      - name: RESOURCE
        value: {{ quote .Values.sidecar.dashboards.resource }}
      {{- with .Values.sidecar.enableUniqueFilenames }}
      - name: UNIQUE_FILENAMES
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.dashboards.searchNamespace }}
      - name: NAMESPACE
        value: "{{ tpl (. | join ",") $root }}"
      {{- end }}
      {{- with .Values.sidecar.skipTlsVerify }}
      - name: SKIP_TLS_VERIFY
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.dashboards.folderAnnotation }}
      - name: FOLDER_ANNOTATION
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.dashboards.script }}
      - name: SCRIPT
        value: "{{ . }}"
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_USER) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_USERNAME
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.userKey | default "admin-user" }}
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_PASSWORD) (not .Values.env.GF_SECURITY_ADMIN_PASSWORD__FILE) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_PASSWORD
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.passwordKey | default "admin-password" }}
      {{- end }}
      {{- if not .Values.sidecar.dashboards.skipReload }}
      - name: REQ_URL
        value: {{ .Values.sidecar.dashboards.reloadURL }}
      - name: REQ_METHOD
        value: POST
      {{- end }}
      {{- if .Values.sidecar.dashboards.watchServerTimeout }}
      {{- if ne .Values.sidecar.dashboards.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.dashboards.watchServerTimeout with .Values.sidecar.dashboards.watchMethod %s" .Values.sidecar.dashboards.watchMethod) }}
      {{- end }}
      - name: WATCH_SERVER_TIMEOUT
        value: "{{ .Values.sidecar.dashboards.watchServerTimeout }}"
      {{- end }}
      {{- if .Values.sidecar.dashboards.watchClientTimeout }}
      {{- if ne .Values.sidecar.dashboards.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.dashboards.watchClientTimeout with .Values.sidecar.dashboards.watchMethod %s" .Values.sidecar.dashboards.watchMethod) }}
      {{- end }}
      - name: WATCH_CLIENT_TIMEOUT
        value: {{ .Values.sidecar.dashboards.watchClientTimeout | quote }}
      {{- end }}
    {{- with .Values.sidecar.livenessProbe }}
    livenessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.readinessProbe }}
    readinessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: sc-dashboard-volume
        mountPath: {{ .Values.sidecar.dashboards.folder | quote }}
      {{- with .Values.sidecar.dashboards.extraMounts }}
      {{- toYaml . | trim | nindent 6 }}
      {{- end }}
{{- end}}
{{- if .Values.sidecar.datasources.enabled }}
  - name: {{ include "grafana.name" . }}-sc-datasources
    {{- if .Values.sidecar.image.sha }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}@sha256:{{ .Values.sidecar.image.sha }}"
    {{- else }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.sidecar.imagePullPolicy }}
    env:
      {{- range $key, $value := .Values.sidecar.datasources.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- if .Values.sidecar.datasources.ignoreAlreadyProcessed }}
      - name: IGNORE_ALREADY_PROCESSED
        value: "true"
      {{- end }}
      - name: METHOD
        value: {{ .Values.sidecar.datasources.watchMethod }}
      - name: LABEL
        value: "{{ .Values.sidecar.datasources.label }}"
      {{- with .Values.sidecar.datasources.labelValue }}
      - name: LABEL_VALUE
        value: {{ quote . }}
      {{- end }}
      {{- if or .Values.sidecar.logLevel .Values.sidecar.datasources.logLevel }}
      - name: LOG_LEVEL
        value: {{ default .Values.sidecar.logLevel .Values.sidecar.datasources.logLevel }}
      {{- end }}
      - name: FOLDER
        value: "/etc/grafana/provisioning/datasources"
      - name: RESOURCE
        value: {{ quote .Values.sidecar.datasources.resource }}
      {{- with .Values.sidecar.enableUniqueFilenames }}
      - name: UNIQUE_FILENAMES
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.datasources.searchNamespace }}
      - name: NAMESPACE
        value: "{{ tpl (. | join ",") $root }}"
      {{- end }}
      {{- if .Values.sidecar.skipTlsVerify }}
      - name: SKIP_TLS_VERIFY
        value: "{{ .Values.sidecar.skipTlsVerify }}"
      {{- end }}
      {{- if .Values.sidecar.datasources.script }}
      - name: SCRIPT
        value: "{{ .Values.sidecar.datasources.script }}"
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_USER) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_USERNAME
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.userKey | default "admin-user" }}
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_PASSWORD) (not .Values.env.GF_SECURITY_ADMIN_PASSWORD__FILE) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_PASSWORD
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.passwordKey | default "admin-password" }}
      {{- end }}
      {{- if not .Values.sidecar.datasources.skipReload }}
      - name: REQ_URL
        value: {{ .Values.sidecar.datasources.reloadURL }}
      - name: REQ_METHOD
        value: POST
      {{- end }}
      {{- if .Values.sidecar.datasources.watchServerTimeout }}
      {{- if ne .Values.sidecar.datasources.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.datasources.watchServerTimeout with .Values.sidecar.datasources.watchMethod %s" .Values.sidecar.datasources.watchMethod) }}
      {{- end }}
      - name: WATCH_SERVER_TIMEOUT
        value: "{{ .Values.sidecar.datasources.watchServerTimeout }}"
      {{- end }}
      {{- if .Values.sidecar.datasources.watchClientTimeout }}
      {{- if ne .Values.sidecar.datasources.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.datasources.watchClientTimeout with .Values.sidecar.datasources.watchMethod %s" .Values.sidecar.datasources.watchMethod) }}
      {{- end }}
      - name: WATCH_CLIENT_TIMEOUT
        value: "{{ .Values.sidecar.datasources.watchClientTimeout }}"
      {{- end }}
    {{- with .Values.sidecar.livenessProbe }}
    livenessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.readinessProbe }}
    readinessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: sc-datasources-volume
        mountPath: "/etc/grafana/provisioning/datasources"
{{- end}}
{{- if .Values.sidecar.notifiers.enabled }}
  - name: {{ include "grafana.name" . }}-sc-notifiers
    {{- if .Values.sidecar.image.sha }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}@sha256:{{ .Values.sidecar.image.sha }}"
    {{- else }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.sidecar.imagePullPolicy }}
    env:
      {{- range $key, $value := .Values.sidecar.notifiers.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- if .Values.sidecar.notifiers.ignoreAlreadyProcessed }}
      - name: IGNORE_ALREADY_PROCESSED
        value: "true"
      {{- end }}
      - name: METHOD
        value: {{ .Values.sidecar.notifiers.watchMethod }}
      - name: LABEL
        value: "{{ .Values.sidecar.notifiers.label }}"
      {{- with .Values.sidecar.notifiers.labelValue }}
      - name: LABEL_VALUE
        value: {{ quote . }}
      {{- end }}
      {{- if or .Values.sidecar.logLevel .Values.sidecar.notifiers.logLevel }}
      - name: LOG_LEVEL
        value: {{ default .Values.sidecar.logLevel .Values.sidecar.notifiers.logLevel }}
      {{- end }}
      - name: FOLDER
        value: "/etc/grafana/provisioning/notifiers"
      - name: RESOURCE
        value: {{ quote .Values.sidecar.notifiers.resource }}
      {{- if .Values.sidecar.enableUniqueFilenames }}
      - name: UNIQUE_FILENAMES
        value: "{{ .Values.sidecar.enableUniqueFilenames }}"
      {{- end }}
      {{- with .Values.sidecar.notifiers.searchNamespace }}
      - name: NAMESPACE
        value: "{{ tpl (. | join ",") $root }}"
      {{- end }}
      {{- with .Values.sidecar.skipTlsVerify }}
      - name: SKIP_TLS_VERIFY
        value: "{{ . }}"
      {{- end }}
      {{- if .Values.sidecar.notifiers.script }}
      - name: SCRIPT
        value: "{{ .Values.sidecar.notifiers.script }}"
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_USER) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_USERNAME
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.userKey | default "admin-user" }}
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_PASSWORD) (not .Values.env.GF_SECURITY_ADMIN_PASSWORD__FILE) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_PASSWORD
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.passwordKey | default "admin-password" }}
      {{- end }}
      {{- if not .Values.sidecar.notifiers.skipReload }}
      - name: REQ_URL
        value: {{ .Values.sidecar.notifiers.reloadURL }}
      - name: REQ_METHOD
        value: POST
      {{- end }}
      {{- if .Values.sidecar.notifiers.watchServerTimeout }}
      {{- if ne .Values.sidecar.notifiers.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.notifiers.watchServerTimeout with .Values.sidecar.notifiers.watchMethod %s" .Values.sidecar.notifiers.watchMethod) }}
      {{- end }}
      - name: WATCH_SERVER_TIMEOUT
        value: "{{ .Values.sidecar.notifiers.watchServerTimeout }}"
      {{- end }}
      {{- if .Values.sidecar.notifiers.watchClientTimeout }}
      {{- if ne .Values.sidecar.notifiers.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.notifiers.watchClientTimeout with .Values.sidecar.notifiers.watchMethod %s" .Values.sidecar.notifiers.watchMethod) }}
      {{- end }}
      - name: WATCH_CLIENT_TIMEOUT
        value: "{{ .Values.sidecar.notifiers.watchClientTimeout }}"
      {{- end }}
    {{- with .Values.sidecar.livenessProbe }}
    livenessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.readinessProbe }}
    readinessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: sc-notifiers-volume
        mountPath: "/etc/grafana/provisioning/notifiers"
{{- end}}
{{- if .Values.sidecar.plugins.enabled }}
  - name: {{ include "grafana.name" . }}-sc-plugins
    {{- if .Values.sidecar.image.sha }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}@sha256:{{ .Values.sidecar.image.sha }}"
    {{- else }}
    image: "{{ .Values.sidecar.image.repository }}:{{ .Values.sidecar.image.tag }}"
    {{- end }}
    imagePullPolicy: {{ .Values.sidecar.imagePullPolicy }}
    env:
      {{- range $key, $value := .Values.sidecar.plugins.env }}
      - name: "{{ $key }}"
        value: "{{ $value }}"
      {{- end }}
      {{- if .Values.sidecar.plugins.ignoreAlreadyProcessed }}
      - name: IGNORE_ALREADY_PROCESSED
        value: "true"
      {{- end }}
      - name: METHOD
        value: {{ .Values.sidecar.plugins.watchMethod }}
      - name: LABEL
        value: "{{ .Values.sidecar.plugins.label }}"
      {{- if .Values.sidecar.plugins.labelValue }}
      - name: LABEL_VALUE
        value: {{ quote .Values.sidecar.plugins.labelValue }}
      {{- end }}
      {{- if or .Values.sidecar.logLevel .Values.sidecar.plugins.logLevel }}
      - name: LOG_LEVEL
        value: {{ default .Values.sidecar.logLevel .Values.sidecar.plugins.logLevel }}
      {{- end }}
      - name: FOLDER
        value: "/etc/grafana/provisioning/plugins"
      - name: RESOURCE
        value: {{ quote .Values.sidecar.plugins.resource }}
      {{- with .Values.sidecar.enableUniqueFilenames }}
      - name: UNIQUE_FILENAMES
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.plugins.searchNamespace }}
      - name: NAMESPACE
        value: "{{ tpl (. | join ",") $root }}"
      {{- end }}
      {{- with .Values.sidecar.plugins.script }}
      - name: SCRIPT
        value: "{{ . }}"
      {{- end }}
      {{- with .Values.sidecar.skipTlsVerify }}
      - name: SKIP_TLS_VERIFY
        value: "{{ . }}"
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_USER) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_USERNAME
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.userKey | default "admin-user" }}
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_PASSWORD) (not .Values.env.GF_SECURITY_ADMIN_PASSWORD__FILE) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: REQ_PASSWORD
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.passwordKey | default "admin-password" }}
      {{- end }}
      {{- if not .Values.sidecar.plugins.skipReload }}
      - name: REQ_URL
        value: {{ .Values.sidecar.plugins.reloadURL }}
      - name: REQ_METHOD
        value: POST
      {{- end }}
      {{- if .Values.sidecar.plugins.watchServerTimeout }}
      {{- if ne .Values.sidecar.plugins.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.plugins.watchServerTimeout with .Values.sidecar.plugins.watchMethod %s" .Values.sidecar.plugins.watchMethod) }}
      {{- end }}
      - name: WATCH_SERVER_TIMEOUT
        value: "{{ .Values.sidecar.plugins.watchServerTimeout }}"
      {{- end }}
      {{- if .Values.sidecar.plugins.watchClientTimeout }}
      {{- if ne .Values.sidecar.plugins.watchMethod "WATCH" }}
        {{- fail (printf "Cannot use .Values.sidecar.plugins.watchClientTimeout with .Values.sidecar.plugins.watchMethod %s" .Values.sidecar.plugins.watchMethod) }}
      {{- end }}
      - name: WATCH_CLIENT_TIMEOUT
        value: "{{ .Values.sidecar.plugins.watchClientTimeout }}"
      {{- end }}
    {{- with .Values.sidecar.livenessProbe }}
    livenessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.readinessProbe }}
    readinessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.sidecar.securityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: sc-plugins-volume
        mountPath: "/etc/grafana/provisioning/plugins"
{{- end}}
  - name: {{ .Chart.Name }}
    {{- if .Values.image.sha }}
    image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}@sha256:{{ .Values.image.sha }}"
    {{- else }}
    image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
    {{- end }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    {{- if .Values.command }}
    command:
    {{- range .Values.command }}
      - {{ . | quote }}
    {{- end }}
    {{- end}}
    {{- with .Values.containerSecurityContext }}
    securityContext:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    volumeMounts:
      - name: config
        mountPath: "/etc/grafana/grafana.ini"
        subPath: grafana.ini
      {{- if .Values.ldap.enabled }}
      - name: ldap
        mountPath: "/etc/grafana/ldap.toml"
        subPath: ldap.toml
      {{- end }}
      {{- range .Values.extraConfigmapMounts }}
      - name: {{ tpl .name $root }}
        mountPath: {{ tpl .mountPath $root }}
        subPath: {{ (tpl .subPath $root) | default "" }}
        readOnly: {{ .readOnly }}
      {{- end }}
      - name: storage
        mountPath: "/var/lib/grafana"
        {{- with .Values.persistence.subPath }}
        subPath: {{ tpl . $root }}
        {{- end }}
      {{- with .Values.dashboards }}
      {{- range $provider, $dashboards := . }}
      {{- range $key, $value := $dashboards }}
      {{- if (or (hasKey $value "json") (hasKey $value "file")) }}
      - name: dashboards-{{ $provider }}
        mountPath: "/var/lib/grafana/dashboards/{{ $provider }}/{{ $key }}.json"
        subPath: "{{ $key }}.json"
      {{- end }}
      {{- end }}
      {{- end }}
      {{- end }}
      {{- with .Values.dashboardsConfigMaps }}
      {{- range (keys . | sortAlpha) }}
      - name: dashboards-{{ . }}
        mountPath: "/var/lib/grafana/dashboards/{{ . }}"
      {{- end }}
      {{- end }}
      {{- with .Values.datasources }}
      {{- range (keys . | sortAlpha) }}
      - name: config
        mountPath: "/etc/grafana/provisioning/datasources/{{ . }}"
        subPath: {{ . | quote }}
      {{- end }}
      {{- end }}
      {{- with .Values.notifiers }}
      {{- range (keys . | sortAlpha) }}
      - name: config
        mountPath: "/etc/grafana/provisioning/notifiers/{{ . }}"
        subPath: {{ . | quote }}
      {{- end }}
      {{- end }}
      {{- with .Values.alerting }}
      {{- range (keys . | sortAlpha) }}
      - name: config
        mountPath: "/etc/grafana/provisioning/alerting/{{ . }}"
        subPath: {{ . | quote }}
      {{- end }}
      {{- end }}
      {{- with .Values.dashboardProviders }}
      {{- range (keys . | sortAlpha) }}
      - name: config
        mountPath: "/etc/grafana/provisioning/dashboards/{{ . }}"
        subPath: {{ . | quote }}
      {{- end }}
      {{- end }}
      {{- with .Values.sidecar.alerts.enabled }}
      - name: sc-alerts-volume
        mountPath: "/etc/grafana/provisioning/alerting"
      {{- end}}
      {{- if .Values.sidecar.dashboards.enabled }}
      - name: sc-dashboard-volume
        mountPath: {{ .Values.sidecar.dashboards.folder | quote }}
      {{- if .Values.sidecar.dashboards.SCProvider }}
      - name: sc-dashboard-provider
        mountPath: "/etc/grafana/provisioning/dashboards/sc-dashboardproviders.yaml"
        subPath: provider.yaml
      {{- end}}
      {{- end}}
      {{- if .Values.sidecar.datasources.enabled }}
      - name: sc-datasources-volume
        mountPath: "/etc/grafana/provisioning/datasources"
      {{- end}}
      {{- if .Values.sidecar.plugins.enabled }}
      - name: sc-plugins-volume
        mountPath: "/etc/grafana/provisioning/plugins"
      {{- end}}
      {{- if .Values.sidecar.notifiers.enabled }}
      - name: sc-notifiers-volume
        mountPath: "/etc/grafana/provisioning/notifiers"
      {{- end}}
      {{- range .Values.extraSecretMounts }}
      - name: {{ .name }}
        mountPath: {{ .mountPath }}
        readOnly: {{ .readOnly }}
        subPath: {{ .subPath | default "" }}
      {{- end }}
      {{- range .Values.extraVolumeMounts }}
      - name: {{ .name }}
        mountPath: {{ .mountPath }}
        subPath: {{ .subPath | default "" }}
        readOnly: {{ .readOnly }}
      {{- end }}
      {{- range .Values.extraEmptyDirMounts }}
      - name: {{ .name }}
        mountPath: {{ .mountPath }}
      {{- end }}
    ports:
      - name: {{ .Values.podPortName }}
        containerPort: {{ .Values.service.targetPort }}
        protocol: TCP
      - name: {{ .Values.gossipPortName }}-tcp
        containerPort: 9094
        protocol: TCP
      - name: {{ .Values.gossipPortName }}-udp
        containerPort: 9094
        protocol: UDP
    env:
      - name: POD_IP
        valueFrom:
          fieldRef:
            fieldPath: status.podIP
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_USER) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: GF_SECURITY_ADMIN_USER
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.userKey | default "admin-user" }}
      {{- end }}
      {{- if and (not .Values.env.GF_SECURITY_ADMIN_PASSWORD) (not .Values.env.GF_SECURITY_ADMIN_PASSWORD__FILE) (not .Values.env.GF_SECURITY_DISABLE_INITIAL_ADMIN_CREATION) }}
      - name: GF_SECURITY_ADMIN_PASSWORD
        valueFrom:
          secretKeyRef:
            name: {{ (tpl .Values.admin.existingSecret .) | default (include "grafana.fullname" .) }}
            key: {{ .Values.admin.passwordKey | default "admin-password" }}
      {{- end }}
      {{- if .Values.plugins }}
      - name: GF_INSTALL_PLUGINS
        valueFrom:
          configMapKeyRef:
            name: {{ include "grafana.fullname" . }}
            key: plugins
      {{- end }}
      {{- if .Values.smtp.existingSecret }}
      - name: GF_SMTP_USER
        valueFrom:
          secretKeyRef:
            name: {{ .Values.smtp.existingSecret }}
            key: {{ .Values.smtp.userKey | default "user" }}
      - name: GF_SMTP_PASSWORD
        valueFrom:
          secretKeyRef:
            name: {{ .Values.smtp.existingSecret }}
            key: {{ .Values.smtp.passwordKey | default "password" }}
      {{- end }}
      {{- if .Values.imageRenderer.enabled }}
      - name: GF_RENDERING_SERVER_URL
        value: http://{{ include "grafana.fullname" . }}-image-renderer.{{ include "grafana.namespace" . }}:{{ .Values.imageRenderer.service.port }}/render
      - name: GF_RENDERING_CALLBACK_URL
        value: {{ .Values.imageRenderer.grafanaProtocol }}://{{ include "grafana.fullname" . }}.{{ include "grafana.namespace" . }}:{{ .Values.service.port }}/{{ .Values.imageRenderer.grafanaSubPath }}
      {{- end }}
      - name: GF_PATHS_DATA
        value: {{ (get .Values "grafana.ini").paths.data }}
      - name: GF_PATHS_LOGS
        value: {{ (get .Values "grafana.ini").paths.logs }}
      - name: GF_PATHS_PLUGINS
        value: {{ (get .Values "grafana.ini").paths.plugins }}
      - name: GF_PATHS_PROVISIONING
        value: {{ (get .Values "grafana.ini").paths.provisioning }}
      {{- range $key, $value := .Values.envValueFrom }}
      - name: {{ $key | quote }}
        valueFrom:
          {{- tpl (toYaml $value) $ | nindent 10 }}
      {{- end }}
      {{- range $key, $value := .Values.env }}
      - name: "{{ tpl $key $ }}"
        value: "{{ tpl (print $value) $ }}"
      {{- end }}
    {{- if or .Values.envFromSecret (or .Values.envRenderSecret .Values.envFromSecrets) .Values.envFromConfigMaps }}
    envFrom:
      {{- if .Values.envFromSecret }}
      - secretRef:
          name: {{ tpl .Values.envFromSecret . }}
      {{- end }}
      {{- if .Values.envRenderSecret }}
      - secretRef:
          name: {{ include "grafana.fullname" . }}-env
      {{- end }}
      {{- range .Values.envFromSecrets }}
      - secretRef:
          name: {{ tpl .name $ }}
          optional: {{ .optional | default false }}
      {{- end }}
      {{- range .Values.envFromConfigMaps }}
      - configMapRef:
          name: {{ tpl .name $ }}
          optional: {{ .optional | default false }}
      {{- end }}
    {{- end }}
    {{- with .Values.livenessProbe }}
    livenessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.readinessProbe }}
    readinessProbe:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- with .Values.lifecycleHooks }}
    lifecycle:
      {{- tpl (toYaml .) $root | nindent 6 }}
    {{- end }}
    {{- with .Values.resources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
{{- with .Values.extraContainers }}
  {{- tpl . $ | nindent 2 }}
{{- end }}
{{- with .Values.nodeSelector }}
nodeSelector:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.affinity }}
affinity:
  {{- tpl (toYaml .) $root | nindent 2 }}
{{- end }}
{{- with .Values.topologySpreadConstraints }}
topologySpreadConstraints:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.tolerations }}
tolerations:
  {{- toYaml . | nindent 2 }}
{{- end }}
volumes:
  - name: config
    configMap:
      name: {{ include "grafana.fullname" . }}
  {{- range .Values.extraConfigmapMounts }}
  - name: {{ tpl .name $root }}
    configMap:
      name: {{ tpl .configMap $root }}
      {{- with .items }}
      items:
        {{- toYaml . | nindent 8 }}
      {{- end }}
  {{- end }}
  {{- if .Values.dashboards }}
  {{- range (keys .Values.dashboards | sortAlpha) }}
  - name: dashboards-{{ . }}
    configMap:
      name: {{ include "grafana.fullname" $ }}-dashboards-{{ . }}
  {{- end }}
  {{- end }}
  {{- if .Values.dashboardsConfigMaps }}
  {{- range $provider, $name := .Values.dashboardsConfigMaps }}
  - name: dashboards-{{ $provider }}
    configMap:
      name: {{ tpl $name $root }}
  {{- end }}
  {{- end }}
  {{- if .Values.ldap.enabled }}
  - name: ldap
    secret:
      {{- if .Values.ldap.existingSecret }}
      secretName: {{ .Values.ldap.existingSecret }}
      {{- else }}
      secretName: {{ include "grafana.fullname" . }}
      {{- end }}
      items:
        - key: ldap-toml
          path: ldap.toml
  {{- end }}
  {{- if and .Values.persistence.enabled (eq .Values.persistence.type "pvc") }}
  - name: storage
    persistentVolumeClaim:
      claimName: {{ tpl (.Values.persistence.existingClaim | default (include "grafana.fullname" .)) . }}
  {{- else if and .Values.persistence.enabled (has .Values.persistence.type $sts) }}
  {{/* nothing */}}
  {{- else }}
  - name: storage
    {{- if .Values.persistence.inMemory.enabled }}
    emptyDir:
      medium: Memory
      {{- with .Values.persistence.inMemory.sizeLimit }}
      sizeLimit: {{ . }}
      {{- end }}
    {{- else }}
    emptyDir: {}
    {{- end }}
  {{- end }}
  {{- if .Values.sidecar.alerts.enabled }}
  - name: sc-alerts-volume
    emptyDir:
      {{- with .Values.sidecar.alerts.sizeLimit }}
      sizeLimit: {{ . }}
      {{- else }}
      {}
      {{- end }}
  {{- end }}
  {{- if .Values.sidecar.dashboards.enabled }}
  - name: sc-dashboard-volume
    emptyDir:
      {{- with .Values.sidecar.dashboards.sizeLimit }}
      sizeLimit: {{ . }}
      {{- else }}
      {}
      {{- end }}
  {{- if .Values.sidecar.dashboards.SCProvider }}
  - name: sc-dashboard-provider
    configMap:
      name: {{ include "grafana.fullname" . }}-config-dashboards
  {{- end }}
  {{- end }}
  {{- if .Values.sidecar.datasources.enabled }}
  - name: sc-datasources-volume
    emptyDir:
      {{- with .Values.sidecar.datasources.sizeLimit }}
      sizeLimit: {{ . }}
      {{- else }}
      {}
      {{- end }}
  {{- end }}
  {{- if .Values.sidecar.plugins.enabled }}
  - name: sc-plugins-volume
    emptyDir:
      {{- with .Values.sidecar.plugins.sizeLimit }}
      sizeLimit: {{ . }}
      {{- else }}
      {}
      {{- end }}
  {{- end }}
  {{- if .Values.sidecar.notifiers.enabled }}
  - name: sc-notifiers-volume
    emptyDir:
      {{- with .Values.sidecar.notifiers.sizeLimit }}
      sizeLimit: {{ . }}
      {{- else }}
      {}
      {{- end }}
  {{- end }}
  {{- range .Values.extraSecretMounts }}
  {{- if .secretName }}
  - name: {{ .name }}
    secret:
      secretName: {{ .secretName }}
      defaultMode: {{ .defaultMode }}
      {{- with .items }}
      items:
        {{- toYaml . | nindent 8 }}
      {{- end }}
  {{- else if .projected }}
  - name: {{ .name }}
    projected:
      {{- toYaml .projected | nindent 6 }}
  {{- else if .csi }}
  - name: {{ .name }}
    csi:
      {{- toYaml .csi | nindent 6 }}
  {{- end }}
  {{- end }}
  {{- range .Values.extraVolumeMounts }}
  - name: {{ .name }}
    {{- if .existingClaim }}
    persistentVolumeClaim:
      claimName: {{ .existingClaim }}
    {{- else if .hostPath }}
    hostPath:
      path: {{ .hostPath }}
    {{- else if .csi }}
    csi:
      {{- toYaml .data | nindent 6 }}
    {{- else }}
    emptyDir: {}
    {{- end }}
  {{- end }}
  {{- range .Values.extraEmptyDirMounts }}
  - name: {{ .name }}
    emptyDir: {}
  {{- end }}
  {{- with .Values.extraContainerVolumes }}
  {{- tpl (toYaml .) $root | nindent 2 }}
  {{- end }}
{{- end }}