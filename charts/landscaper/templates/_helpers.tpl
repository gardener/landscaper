{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "landscaper.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "landscaper.fullname" -}}
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
{{- define "landscaper.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "landscaper.labels" -}}
helm.sh/chart: {{ include "landscaper.chart" . }}
{{ include "landscaper.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "landscaper.selectorLabels" -}}
landscaper.gardener.cloud/component: controller
app.kubernetes.io/name: {{ include "landscaper.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "landscaper.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "landscaper.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "landscaper-config" -}}
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

{{- if .Values.landscaper.registryConfig }}
registry:
    oci:
      allowPlainHttp: {{ .Values.landscaper.registryConfig.allowPlainHttpRegistries }}
      insecureSkipVerify: {{ .Values.landscaper.registryConfig.insecureSkipVerify }}
      {{- if .Values.landscaper.registryConfig.secrets }}
      configFiles:
      {{- range $key, $value := .Values.landscaper.registryConfig.secrets }}
      - /app/ls/registry/secrets/{{ $key }}
      {{- end }}
      {{- end }}
      cache:
        path: /app/ls/oci-cache/
        useInMemoryOverlay: {{ .Values.landscaper.registryConfig.cache.useInMemoryOverlay | default false }}
{{ end }}
{{- if .Values.landscaper.metrics }}
metrics:
  port: {{ .Values.landscaper.metrics.port | default 8080 }}
{{- end }}
{{- if .Values.landscaper.crdManagement }}
crdManagement:
    deployCrd: {{ .Values.landscaper.crdManagement.deployCrd }}
    {{- if .Values.landscaper.crdManagement.forceUpdate }}
    forceUpdate: {{ .Values.landscaper.crdManagement.forceUpdate }}
    {{- end }}
{{- end }}

{{- end }}