package service

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/fenole/szmaterlok/service/sse"
	"github.com/fenole/szmaterlok/web"
)

// RouterDependencies holds all configurated dependencies
// for new http router.
type RouterDependencies struct {
	Logger       *logrus.Logger
	SessionStore *SessionCookieStore
	MessageSender
	MessageNotifier
	IDGenerator
	Clock
}

// NewRouter returns new configured chi mux router.
func NewRouter(deps RouterDependencies) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(&LoggerLogFormatter{
		Logger: deps.Logger,
	}))
	r.Use(middleware.Recoverer)

	r.Get("/", HandlerIndex(web.UI))
	r.Post("/login", HandlerLogin(HandlerLoginDependencies{
		StateFactory: DefaultSessionStateFactory(),
		Logger:       deps.Logger,
		SessionStore: deps.SessionStore,
	}))
	r.Post("/logout", HandlerLogout(deps.SessionStore))
	r.With(SessionRequired(deps.SessionStore)).Get("/chat", HandlerChat(web.UI))
	r.With(SessionRequired(deps.SessionStore), sse.Headers).Get("/stream", HandlerStream(deps))
	r.With(SessionRequired(deps.SessionStore)).Post("/message", HandlerSendMessage(HandlerSendMessageDependencies{
		Sender:      deps,
		IDGenerator: deps,
		Clock:       deps,
	}))
	r.Handle("/*", http.FileServer(http.FS(web.Assets)))

	return r
}
