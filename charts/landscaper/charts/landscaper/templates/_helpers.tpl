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

{{- define "landscaper.webhooks.fullname" -}}
{{- include "landscaper.fullname" . }}-webhooks
{{- end }}

{{- define "landscaper.agent.fullname" -}}
{{- include "landscaper.fullname" . }}-agent
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "landscaper.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "landscaper.controller.containerName" -}}
{{- if .Values.controller.containerName -}}
{{- .Values.controller.containerName | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- print "landscaper-controller" }}
{{- end }}
{{- end }}

{{- define "landscaper.webhooks.containerName" -}}
{{- if .Values.webhooksServer.containerName -}}
{{- .Values.webhooksServer.containerName | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- print "landscaper-webhooks" }}
{{- end }}
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

{{- define "landscaper.webhooks.selectorLabels" -}}
landscaper.gardener.cloud/component: webhook-server
app.kubernetes.io/name: {{ include "landscaper.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "landscaper.controller.serviceAccountName" -}}
{{- default "landscaper" .Values.global.serviceAccount.controller.name }}
{{- end }}

{{- define "landscaper.webhooks.serviceAccountName" -}}
{{- default "landscaper-webhooks" .Values.global.serviceAccount.webhooksServer.name }}
{{- end }}

{{- define "landscaper-config" -}}
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

{{- if .Values.landscaper.repositoryContext }}
repositoryContext:
    type: {{ .Values.landscaper.repositoryContext.type }}
    baseUrl: {{ .Values.landscaper.repositoryContext.baseUrl }}
{{- end }}

{{- if .Values.landscaper.controllers }}
controllers:
{{ .Values.landscaper.controllers | toYaml | indent 2 }}
{{- end }}

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

{{- if .Values.landscaper.deployerManagement }}
{{- if .Values.landscaper.deployerManagement.agent.name }}
{{- if gt (len .Values.landscaper.deployerManagement.agent.name) 30 }}
{{- fail "the length of .Values.landscaper.deployerManagement.agent.name may not be greater than 30" }}
{{- end }}
{{- end }}
deployerManagement:
{{ toYaml .Values.landscaper.deployerManagement | indent 2 }}
{{- end -}}

{{- if .Values.landscaper.deployItemTimeouts }}
deployItemTimeouts:
  {{- range $key, $value := .Values.landscaper.deployItemTimeouts }}
  {{ $key }}: {{ $value }}
  {{- end }}
{{- end }}

lsDeployments:
  lsController: "{{- include "landscaper.fullname" . }}"
  webHook: "{{- include "landscaper.webhooks.fullname" . }}"

{{- end }}

{{- define "landscaper-image" -}}
{{- $tag := ( .Values.controller.image.tag | default .Values.image.tag | default .Chart.AppVersion )  -}}
{{- $image :=  dict "repository" .Values.controller.image.repository "tag" $tag  -}}
{{- include "utils-templates.image" $image }}
{{- end -}}

{{- define "landscaper-webhook-image" -}}
{{- $tag := ( .Values.webhooksServer.image.tag | default .Values.image.tag | default .Chart.AppVersion )  -}}
{{- $image :=  dict "repository" .Values.webhooksServer.image.repository "tag" $tag  -}}
{{- include "utils-templates.image" $image }}
{{- end -}}

{{- define "utils-templates.image" -}}
{{- if hasPrefix "sha256:" (required "$.tag is required" $.tag) -}}
{{ required "$.repository is required" $.repository }}@{{ required "$.tag is required" $.tag }}
{{- else -}}
{{ required "$.repository is required" $.repository }}:{{ required "$.tag is required" $.tag }}
{{- end -}}
{{- end -}}