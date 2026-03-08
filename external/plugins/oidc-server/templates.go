package main

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	loginTemplate  = template.Must(template.ParseFS(templateFS, "templates/login.html"))
	errorTemplate  = template.Must(template.ParseFS(templateFS, "templates/error.html"))
	logoutTemplate = template.Must(template.ParseFS(templateFS, "templates/logout.html"))
)
