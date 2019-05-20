package main

import (
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/stts-se/weblib/i18n"
)

// TemplateData used to execute a html/template/Template. If properly used, it will fill in the correct i18n values in the template
type TemplateData struct {
	Loc  *i18n.I18N
	Data interface{}
}

func templateFromName(templateName string) string {
	return filepath.Join("templates", fmt.Sprintf("%s.html", templateName))
}

var templates = template.Must(template.ParseFiles(
	templateFromName("login"),
	templateFromName("logout"),
	templateFromName("invite"),
	templateFromName("signup"),
))
