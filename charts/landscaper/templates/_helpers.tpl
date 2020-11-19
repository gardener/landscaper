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

registries:
  components:
  {{- if .Values.landscaper.registryConfig.components }}
    oci:
      allowPlainHttp: {{ .Values.landscaper.registryConfig.components.allowPlainHttpRegistries }}
      {{- if .Values.landscaper.registryConfig.components.secrets }}
      configFiles:
      {{- range $key, $value := .Values.landscaper.registryConfig.components.secrets }}
      - /app/ls/registry/components/{{ $key }}
      {{- end }}
      {{- end }}
  {{- end }}
  blueprints:
    {{- if .Values.landscaper.registryConfig.blueprints }}
    oci:
      allowPlainHttp: {{ .Values.landscaper.registryConfig.blueprints.allowPlainHttpRegistries }}
      {{- if .Values.landscaper.registryConfig.blueprints.secrets }}
      configFiles:
      {{- range $key, $value := .Values.landscaper.registryConfig.blueprints.secrets }}
      - /app/ls/registry/blueprints/{{ $key }}
      {{- end }}
      {{- end }}
    {{- end }}
{{- end }}