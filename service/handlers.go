package service

import (
	"context"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/fenole/szmaterlok/service/sse"
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

// MessageSent is SSE event type for message sent event.
const MessageSent = "message-sent"

// MessageAuthor is author of single message sent.
type MessageAuthor struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

// EventSentMessage is model for event of single sent message
// by client to all listeners.
type EventSentMessage struct {
	ID      string        `json:"id"`
	From    MessageAuthor `json:"from"`
	Content string        `json:"content"`
	SentAt  time.Time     `json:"sentAt"`
}

// MessageSender sends clients event messages.
type MessageSender interface {
	// SendMessage sends event message to all subscribers. This
	// method is supposed to be blocking.
	SendMessage(ctx context.Context, evt EventSentMessage)
}

// MessageSubscribeRequest holds arguments for subscribe
// method of MessageNotifier.
type MessageSubscribeRequest struct {
	// ID is chat (channel, user or any other chat entity) ID.
	ID string

	// RequestID is unique request ID. One client, with the same ID,
	// can have multiple request IDs.
	RequestID string

	// Channel for sending SSE events.
	Channel chan<- sse.Event
}

// MessageNotifier sends SSE events notifications to client.
type MessageNotifier interface {
	// Subscribe given ID for SSE events. Returns unsubscribe func.
	Subscribe(ctx context.Context, args MessageSubscribeRequest) func()
}

// HandlerStream is SSE event stream handler, which sends event notifications
// to clients. It requires authentication.
//
// See SessionRequired middleware.
func HandlerStream(notifier MessageNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		state := SessionContextState(ctx)
		if state == nil {
			jsonResponse(w, http.StatusForbidden, responseWrapper{
				Error: errorResponse{
					Code:    http.StatusForbidden,
					Message: "Event stream requires authentication.",
				},
			})
			return
		}

		// Make sure that the writer supports flushing.
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		evts := make(chan sse.Event)
		unsubscribe := notifier.Subscribe(ctx, MessageSubscribeRequest{
			ID:        state.ID,
			RequestID: middleware.GetReqID(ctx),
			Channel:   evts,
		})
		defer unsubscribe()

		for {
			select {
			case evt := <-evts:
				if err := sse.Encode(w, evt); err != nil {
					jsonResponse(w, http.StatusInternalServerError, responseWrapper{
						Error: errorResponse{
							Code:    http.StatusInternalServerError,
							Message: "Failed to encode event stream message.",
						},
					})
					return
				}

				// Flush the data immediatly instead of buffering it for later.
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

// HandlerLoginDependencies holds behavioral dependencies for
// http handler for sending messages.
type HandlerSendMessageDependencies struct {
	Sender MessageSender
	IDGenerator
	Clock
}

// HandlerSendMessage handles sending message to all current listeners.
func HandlerSendMessage(deps HandlerSendMessageDependencies) http.HandlerFunc {
	type request struct {
		Content string `json:"content"`
	}
	type response struct {
		ID string `json:"id"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		state := SessionContextState(ctx)
		if state == nil {
			jsonResponse(w, http.StatusForbidden, responseWrapper{
				Error: errorResponse{
					Code:    http.StatusForbidden,
					Message: "Sending messages requires authentication.",
				},
			})
			return
		}

		req := &request{}

		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			jsonResponse(w, http.StatusBadRequest, responseWrapper{
				Error: errorResponse{
					Code:    http.StatusBadRequest,
					Message: "Failed to parse body.",
				},
			})
			return
		}

		messageID := deps.GenerateID()
		go deps.Sender.SendMessage(ctx, EventSentMessage{
			ID: messageID,
			From: MessageAuthor{
				ID:       state.ID,
				Nickname: state.Nickname,
			},
			Content: req.Content,
			SentAt:  deps.Now(),
		})

		jsonResponse(w, http.StatusAccepted, responseWrapper{
			Data: response{
				ID: messageID,
			},
		})
	}
}
