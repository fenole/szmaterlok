package service

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"
)

// HandlerIndex renders main page of szmaterlok.
func HandlerIndex(f fs.FS) http.HandlerFunc {
	var tmpl *template.Template
	once := &sync.Once{}

	return func(w http.ResponseWriter, R *http.Request) {
		once.Do(func() {
			tmpl = template.Must(template.ParseFS(f, "ui/layout.html", "ui/index.html"))
		})

		w.WriteHeader(http.StatusOK)
		if err := tmpl.ExecuteTemplate(w, "layout", nil); err != nil {
			http.Error(w, "failed to parse delivered html template", http.StatusInternalServerError)
			return
		}
	}
}
