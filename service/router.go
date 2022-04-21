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
	Bridge       *Bridge

	MaximumMessageSize int

	AllChatUsersStore
	MessageNotifier
	IDGenerator
	Clock
}

// NewRouter returns new configured chi mux router.
func NewRouter(deps RouterDependencies) *chi.Mux {
	r := chi.NewRouter()

	sessionRequired := SessionRequired(deps.SessionStore)

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
	r.With(sessionRequired).Get("/chat", HandlerChat(web.UI))
	r.With(LastEventIDMiddleware, sessionRequired, sse.Headers).Get("/stream", HandlerStream(HandlerStreamDependencies{
		MessageNotifier: &EventAnnouncer{
			MessageNotifier: deps.MessageNotifier,
			UserJoinProducer: &BridgeEventProducer[EventUserJoin]{
				EventBridge: deps.Bridge,
				Type:        BridgeUserJoin,
				Log:         deps.Logger,
				Clock:       deps,
			},
			UserLeftProducer: &BridgeEventProducer[EventUserLeft]{
				EventBridge: deps.Bridge,
				Type:        BridgeUserLeft,
				Log:         deps.Logger,
				Clock:       deps,
			},
			Clock:       deps,
			IDGenerator: deps,
		},
		IDGenerator: deps,
		Clock:       deps,
	}))
	r.With(sessionRequired).Post("/message", HandlerSendMessage(HandlerSendMessageDependencies{
		Sender: &BridgeEventProducer[EventSentMessage]{
			EventBridge: deps.Bridge,
			Type:        BridgeMessageSent,
			Log:         deps.Logger,
			Clock:       deps,
		},
		IDGenerator:    deps,
		Clock:          deps,
		MaxMessageSize: deps.MaximumMessageSize,
	}))
	r.With(sessionRequired).Get("/users", HandlerOnlineUsers(deps.Logger, deps))
	r.Handle("/*", http.FileServer(http.FS(web.Assets)))

	return r
}
