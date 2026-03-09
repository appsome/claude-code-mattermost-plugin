{{/*
Expand the name of the chart.
*/}}
{{- define "claude-code-bridge.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "claude-code-bridge.fullname" -}}
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
{{- define "claude-code-bridge.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "claude-code-bridge.labels" -}}
helm.sh/chart: {{ include "claude-code-bridge.chart" . }}
{{ include "claude-code-bridge.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: claude-code-mattermost-plugin
{{- end }}

{{/*
Selector labels
*/}}
{{- define "claude-code-bridge.selectorLabels" -}}
app.kubernetes.io/name: {{ include "claude-code-bridge.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: bridge
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "claude-code-bridge.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "claude-code-bridge.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Check if API key is configured (either directly or via existing secret)
*/}}
{{- define "claude-code-bridge.apiKeyConfigured" -}}
{{- if or .Values.anthropicApiKey .Values.existingSecret.name -}}
true
{{- end }}
{{- end }}

{{/*
Get the secret name for Anthropic API key
*/}}
{{- define "claude-code-bridge.secretName" -}}
{{- if .Values.existingSecret.name }}
{{- .Values.existingSecret.name }}
{{- else }}
{{- include "claude-code-bridge.fullname" . }}
{{- end }}
{{- end }}

{{/*
Get the secret key for Anthropic API key
*/}}
{{- define "claude-code-bridge.secretKey" -}}
{{- if .Values.existingSecret.name }}
{{- .Values.existingSecret.key | default "ANTHROPIC_API_KEY" }}
{{- else }}
{{- "ANTHROPIC_API_KEY" }}
{{- end }}
{{- end }}
