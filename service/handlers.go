package service

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

// HandlerIndex renders main page of szmaterlok.
func HandlerIndex(f fs.FS) http.HandlerFunc {
	var tmpl *template.Template
	once := &sync.Once{}

	return func(w http.ResponseWriter, r *http.Request) {
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

// HandlerChat renders chat application view of szmaterlok.
func HandlerChat(f fs.FS) http.HandlerFunc {
	var tmpl *template.Template
	once := &sync.Once{}

	return func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			tmpl = template.Must(template.ParseFS(f, "ui/layout.html", "ui/chat.html"))
		})

		w.WriteHeader(http.StatusOK)
		if err := tmpl.ExecuteTemplate(w, "layout", nil); err != nil {
			http.Error(w, "failed to parse delivered html template", http.StatusInternalServerError)
			return
		}
	}
}

// HandlerLoginDependencies holds behavioral dependencies for
// login http handler.
type HandlerLoginDependencies struct {
	StateFactory *SessionStateFactory
	Logger       *logrus.Logger
	SessionStore *SessionCookieStore
}

func HandlerLogin(deps HandlerLoginDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nickname := r.FormValue("nickname")
		if nickname == "" {
			http.Error(w, "Nickname cannot be empty.", http.StatusBadRequest)
			return
		}

		state := deps.StateFactory.MakeState(nickname)
		if err := deps.SessionStore.SaveSessionState(w, state); err != nil {
			http.Error(w, "Failed to save session state.", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/chat", http.StatusSeeOther)
	}
}

func HandlerLogout(cs *SessionCookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cs.ClearState(w)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
