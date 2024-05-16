{{/* SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

 SPDX-License-Identifier: Apache-2.0
*/}}

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
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "landscaper.controller.serviceAccountName" -}}
{{- default "landscaper" .Values.global.serviceAccount.controller.name }}
{{- end }}

{{- define "landscaper.webhooksServer.serviceAccountName" -}}
{{- default "landscaper-webhooks" .Values.global.serviceAccount.webhooksServer.name }}
{{- end }}

{{- define "landscaper.user.serviceAccountName" -}}
{{- default "landscaper-user" .Values.global.serviceAccount.user.name }}
{{- end }}

{{/*
Create the name of the of the aggregation cluster roles
*/}}
{{- define "landscaper.aggregation.admin.clusterRoleName" -}}
{{- default "landscaper:aggregate-to-admin" .Values.aggregation.admin.name }}
{{- end }}

{{- define "landscaper.aggregation.view.clusterRoleName" -}}
{{- default "landscaper:aggregate-to-view" .Values.aggregation.view.name}}
{{- end }}

