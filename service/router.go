package service

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"

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
}

// NewRouter returns new configured chi mux router.
func NewRouter(deps RouterDependencies) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(&LoggerLogFormatter{
		Logger: deps.Logger,
	}))
	r.Use(middleware.Recoverer)

	r.With(sse.Headers).Get("/stream", func(w http.ResponseWriter, r *http.Request) {
		type res struct {
			Message string `json:"msg"`
			Time    int64  `json:"sendAt"`
		}

		// Make sure that the writer supports flushing.
		flusher, ok := w.(http.Flusher)

		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		tick := time.Tick(time.Second * 2)

		for {
			select {
			case <-tick:
				eventData, err := json.Marshal(res{
					Message: "hello world!",
					Time:    time.Now().UnixNano(),
				})

				if err != nil {
					return
				}
				sse.Encode(w, sse.Event{
					Type: "test",
					ID:   strconv.Itoa(rand.Int()),
					Data: eventData,
				})

				// Flush the data immediatly instead of buffering it for later.
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	r.Handle("/", HandlerIndex(web.UI))
	r.With(SessionRequired(deps.SessionStore)).Handle("/chat", HandlerChat(web.UI))
	r.Handle("/*", http.FileServer(http.FS(web.Assets)))

	return r
}
