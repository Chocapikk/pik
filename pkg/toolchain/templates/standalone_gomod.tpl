module pik-standalone

go 1.25.6

require github.com/Chocapikk/pik {{.Version}}
{{if .ModRoot}}
replace github.com/Chocapikk/pik => {{.ModRoot}}
{{end}}