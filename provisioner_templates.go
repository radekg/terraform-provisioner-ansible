package main

const inventoryTemplateRemote = `{{$top := . -}}
{{range .Hosts -}}
{{.}} ansible_connection=local
{{end}}

{{range .Groups -}}
[{{.}}]
{{range $top.Hosts -}}
{{.}} ansible_connection=local
{{end}}

{{end}}`

const inventoryTemplateLocal = `{{$top := . -}}
{{range .Hosts -}}
{{.}}
{{end}}

{{range .Groups -}}
[{{.}}]
{{range $top.Hosts -}}
{{.}}
{{end}}

{{end}}`
