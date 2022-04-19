package service

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/fenole/szmaterlok/service/sse"
)

type bufferNode struct {
	value *EventSentMessage
	next  *bufferNode
}

// MessageCircularBuffer is thread-safe data structure which
// holds fixed number of events. When buffer is full, push
// overwrites oldest item.
type MessageCircularBuffer struct {
	head *bufferNode
	mtx  *sync.Mutex
}

// NewMessageCircularBuffer returns address of circular buffer
// with given size.
func NewMessageCircularBuffer(size int) *MessageCircularBuffer {
	head := &bufferNode{}

	last := head
	for i := 1; i < size; i++ {
		last.next = &bufferNode{}
		last = last.next
	}
	last.next = head

	return &MessageCircularBuffer{
		mtx:  &sync.Mutex{},
		head: head,
	}
}

// PushEvent appends given sent message event to the circular buffer.
// If buffer is full: push overwrites oldest item.
func (mb *MessageCircularBuffer) PushEvent(ctx context.Context, evt EventSentMessage) {
	mb.mtx.Lock()
	defer mb.mtx.Unlock()

	mb.head.value = &evt
	mb.head = mb.head.next
}

// BufferedEvents returns all of events stored in the buffer.
func (mb *MessageCircularBuffer) BufferedEvents(ctx context.Context) []EventSentMessage {
	mb.mtx.Lock()
	defer mb.mtx.Unlock()

	res := []EventSentMessage{}

	curr := mb.head
	for {
		if curr.value != nil {
			res = append(res, *curr.value)
		}

		if curr.next == mb.head {
			break
		}

		curr = curr.next
	}

	return res
}

// LastMessagesBuffer keeps fixed number of messages that can be
// send to users to give them a little brief overview about current
// discussion.
type LastMessagesBuffer struct {
	buffer *MessageCircularBuffer
	log    *logrus.Logger
}

// NewLastMessagesBuffer returns last message buffer of given size.
func NewLastMessagesBuffer(size int, log *logrus.Logger) *LastMessagesBuffer {
	return &LastMessagesBuffer{
		buffer: NewMessageCircularBuffer(size),
	}
}

func findEventByID(target string, items []EventSentMessage) (int, bool) {
	for i, item := range items {
		if item.ID == target {
			return i, true
		}
	}

	return 0, false
}

// LastMessages returns all messages stored in LastMessagesBuffer that happened
// after event with given last message ID.
func (b *LastMessagesBuffer) LastMessages(ctx context.Context, lastMessageID string) []EventSentMessage {
	items := b.buffer.BufferedEvents(ctx)

	if lastMessageID == "" {
		return items
	}

	target, ok := findEventByID(lastMessageID, items)
	if !ok {
		return items
	}

	res := []EventSentMessage{}
	for i, item := range items {
		if i == target {
			continue
		}
		if item.SentAt.After(items[i].SentAt) {
			continue
		}

		res = append(res, item)
	}

	return res
}

// EventHook listens for message-sent events and appends them to the
// last messages circular buffer.
func (b *LastMessagesBuffer) EventHook(ctx context.Context, evt BridgeEvent) {
	evtData := EventSentMessage{}

	if err := json.Unmarshal(evt.Data, &evtData); err != nil {
		b.log.WithFields(logrus.Fields{
			"scope":   "StateUserJoinHook",
			"reqID":   evt.Headers.Get(bridgeRequestIDHeaderVar),
			"eventID": evt.ID,
			"error":   err.Error(),
		}).Errorln("Failed to unmarshal EventSentMessage data.")
		return
	}

	b.buffer.PushEvent(ctx, evtData)
}

// MessageNotifierWithBuffer is adapter for MessageNotifier which
// sends messages from last messages buffer to subscribed clients.
type MessageNotifierWithBuffer struct {
	Notifier MessageNotifier
	Buffer   *LastMessagesBuffer
	Logger   *logrus.Logger
}

type contextLastEventIDKey int

const lastEventIDKey contextLastEventIDKey = 1

// ContextWithLastEventID stores given event ID within the context.
func ContextWithLastEventID(ctx context.Context, lastEventID string) context.Context {
	return context.WithValue(ctx, lastEventIDKey, lastEventID)
}

func contextLastEventID(ctx context.Context) string {
	res, ok := ctx.Value(lastEventIDKey).(string)
	if !ok {
		return ""
	}
	return res
}

// Subscribe given ID for SSE events. Returns unsubscribe func.
func (m *MessageNotifierWithBuffer) Subscribe(ctx context.Context, args MessageSubscribeRequest) func() {
	lastEventID := contextLastEventID(ctx)

	buffered := m.Buffer.LastMessages(ctx, lastEventID)
	tmpChan := make(chan sse.Event, len(buffered))

	for _, msg := range buffered {
		b, err := json.Marshal(msg)
		if err != nil {
			m.Logger.WithField("eventID", msg.ID).Error("Failed to marshal event.")
			continue
		}

		tmpChan <- sse.Event{
			Type: MessageSent,
			Data: b,
			ID:   msg.ID,
		}
	}
	close(tmpChan)

	// transientChan is bridge between channel created by client
	// and channel created in this method. This way we can be sure
	// that client will first receive buffered events and then
	transientChan := make(chan sse.Event)

	go func() {
		m.Logger.WithFields(logrus.Fields{
			"reqID": args.RequestID,
			"subID": args.ID,
		}).Trace("Transient goroutine has started.")

		for msg := range tmpChan {
			args.Channel <- msg
		}

		m.Logger.WithFields(logrus.Fields{
			"reqID": args.RequestID,
			"subID": args.ID,
		}).Trace("Buffered messages have been sent.")

		for msg := range transientChan {
			args.Channel <- msg
		}

		m.Logger.WithFields(logrus.Fields{
			"reqID": args.RequestID,
			"subID": args.ID,
		}).Trace("Transient goroutine has been terminated.")
	}()

	unsubscribe := m.Notifier.Subscribe(ctx, MessageSubscribeRequest{
		ID:        args.ID,
		RequestID: args.RequestID,
		Channel:   transientChan,
	})

	wrappedUnsubscribe := func() {
		unsubscribe()
		close(transientChan)
	}
	return wrappedUnsubscribe
}

func requestsLastEventID(h http.Header) string {
	return h.Get("Last-Event-ID")
}

// LastEventIDMiddleware injects Last-Event-ID header value into the requests
// context.
func LastEventIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastEventID := requestsLastEventID(r.Header)
		newCtx := ContextWithLastEventID(r.Context(), lastEventID)
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}
