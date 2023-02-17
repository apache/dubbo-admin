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