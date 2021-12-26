package main

import (
	_ "embed"
	"html/template"
	"net/http"

	"go.uber.org/zap"
)

//go:embed templates/status.html
var statusPageTemplateStr string
var statusPageTemplate = template.Must(template.New("status.html").Parse(statusPageTemplateStr))

//go:embed templates/errors.html
var errorPageTemplateStr string
var errorPageTemplate = template.Must(template.New("errors.html").Parse(errorPageTemplateStr))

//go:embed static/icon.svg
var iconSvg []byte

// StatusPage is a http.Handler for status page.
type StatusPage struct {
	StatusManager *StatusManager
}

func (s StatusPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		err = errorPageTemplate.Execute(w, "Method not allowed")
	} else if r.URL.Path == "/favicon.ico" {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, err = w.Write(iconSvg)
	} else if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		err = errorPageTemplate.Execute(w, "Not found")
	} else {
		ss := s.StatusManager.Status()
		err = statusPageTemplate.Execute(w, map[string]interface{}{
			"Status": ss,
		})
	}

	if err != nil {
		zap.L().Error(
			"failed to render page",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
	}
}
