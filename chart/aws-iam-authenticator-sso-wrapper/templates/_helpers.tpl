{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{define "__CHART__.name"}}{{default "__CHART__" .Values.nameOverride | trunc 63 | trimSuffix "-" }}{{end}}

{{/*
Create a default fully qualified app name.

We truncate at 63 chars because some Kubernetes name fields are limited to this
(by the DNS naming spec).
*/}}
{{define "__CHART__.fullname"}}
{{- $name := default "__CHART__" .Values.nameOverride -}}
{{printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{end}}
