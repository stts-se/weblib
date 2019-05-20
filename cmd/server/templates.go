package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/stts-se/weblib"
)

func templateName2Path(templateName string) string {
	return filepath.Join("templates", fmt.Sprintf("%s.html", templateName))
}

var templates = template.Must(template.ParseFiles(
	templateName2Path("login"),
	templateName2Path("logout"),
	templateName2Path("invite"),
	templateName2Path("signup"),
))

func parseTemplate(templateName string, data interface{}) (*template.Template, error) {
	tpl, err := weblib.ReadFile(templateName2Path(templateName))
	if err != nil {
		return &template.Template{}, err
	}
	t, err := template.New(templateName).Parse(tpl)
	if err != nil {
		return &template.Template{}, err
	}
	return t, nil
}

func executeTemplate(templateName string, data interface{}, w http.ResponseWriter) error {
	t, err := parseTemplate(templateName, data)
	if err != nil {
		return err
	}
	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return nil
}
