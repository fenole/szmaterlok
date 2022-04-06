package service

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/fenole/szmaterlok/service/sse"
)

// BridgeEventType represents event name by which
// events can be grouped by. Events of one type should
// have the same data scheme.
type BridgeEventType string

// BridgeEventGlob matches all event types. It can be used to
// listen to all possible events.
const BridgeEventGlob BridgeEventType = "*"

// BridgeHeaders store event store metadata.
type BridgeHeaders map[string]string

// Get returns value for given key. If there is no value
// associated with given key, get returns empty string.
func (h BridgeHeaders) Get(key string) string {
	res, ok := h[key]
	if !ok {
		return ""
	}
	return res
}

// BridgeEvent is single event data model and commont
// interface for all events.
type BridgeEvent struct {
	// Name is event type.
	Name BridgeEventType `json:"type"`

	// ID is unique event identifier.
	ID string `json:"id"`

	// CreatedAt is date of event creation expressed
	// as unix epoch.
	CreatedAt int64 `json:"createdAt"`

	// Headers holds event metadata such as. For
	// example: one's could use headers to store
	// event data content type.
	Headers BridgeHeaders `json:"headers"`

	// Data sent or stored with event.
	Data []byte `json:"data"`
}

// BridgeEventHandler implements behaviour for dealing
// with events from szmaterlok event bridge.
type BridgeEventHandler interface {
	// EventHook can implement any generic operation which uses
	// data from BridgeEvent type.
	EventHook(context.Context, BridgeEvent)
}

// BridgeEventHandlerFunc is functional interface of BridgeEventHandler.
type BridgeEventHandlerFunc func(context.Context, BridgeEvent)

func (f BridgeEventHandlerFunc) EventHook(ctx context.Context, evt BridgeEvent) {
	f(ctx, evt)
}

type bridgeEventHandlerComposite []BridgeEventHandler

func (ehc bridgeEventHandlerComposite) EventHook(ctx context.Context, evt BridgeEvent) {
	wg := sync.WaitGroup{}
	wg.Add(len(ehc))
	for _, h := range ehc {
		h := h
		go func() {
			defer wg.Done()
			h.EventHook(ctx, evt)
		}()
	}
	wg.Wait()
}

// Bridge is asynchronous queue for events. It can process
// events from different sources spread all across szmaterlok
// application and handles them with event hooks represented
// as event handlers.
//
// Single event type can have multiple event handlers.
type Bridge struct {
	queue  chan BridgeEvent
	closer chan struct{}

	handler BridgeEventHandler
}

// NewBridge is constructor for event bridge. It returns
// default instance of event bridge.
func NewBridge(ctx context.Context, handler BridgeEventHandler) *Bridge {
	evtChan := make(chan BridgeEvent)
	res := &Bridge{
		queue:   evtChan,
		closer:  make(chan struct{}),
		handler: handler,
	}

	go res.run(ctx)
	return res
}

// SendEvent sends event to event bridge. It blocks, so it's
// a good idea to run it in a separate goroutine.
func (b *Bridge) SendEvent(evt BridgeEvent) {
	b.queue <- evt
}

// Shutdown closes event bridge and waits for current
// events being processed to finish.
func (b *Bridge) Shutdown(ctx context.Context) {
	close(b.queue)

	select {
	case <-b.closer:
		return
	case <-ctx.Done():
		return
	}
}

// goWithWaitGroup is helper for running jobs with the help
// of wait group for further synchronization.
func goWithWaitGroup(wg *sync.WaitGroup, f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}

// run is main event loop of event bridge.
func (b *Bridge) run(ctx context.Context) {
	wg := sync.WaitGroup{}

	// Main processing loop.
	for evt := range b.queue {
		evt := evt

		if b.handler == nil {
			continue
		}

		goWithWaitGroup(&wg, func() {
			b.handler.EventHook(ctx, evt)
		})
	}

	// Wait for all jobs to finish.
	wg.Wait()

	// Send signal to closer and indicate event loop has finished.
	b.closer <- struct{}{}
}

// BridgeEventRouter delegates different event types into
// their associated hook handlers.
type BridgeEventRouter struct {
	hooks map[BridgeEventType]bridgeEventHandlerComposite
}

func NewBridgeEventRouter() *BridgeEventRouter {
	return &BridgeEventRouter{
		hooks: map[BridgeEventType]bridgeEventHandlerComposite{},
	}
}

// Hook adds given event handler to hook list for given event type.
// Given hook will be fired when router receives new event
// with matching event type.
//
// All hooks should be added before mounting event router to bridge.
func (r *BridgeEventRouter) Hook(t BridgeEventType, h BridgeEventHandler) {
	_, ok := r.hooks[t]
	if !ok {
		r.hooks[t] = bridgeEventHandlerComposite{}
	}

	r.hooks[t] = append(r.hooks[t], h)
}

func (r *BridgeEventRouter) EventHook(ctx context.Context, evt BridgeEvent) {
	wg := sync.WaitGroup{}

	globHandler, ok := r.hooks[BridgeEventGlob]
	if ok {
		goWithWaitGroup(&wg, func() {
			globHandler.EventHook(ctx, evt)
		})
	}

	handler, ok := r.hooks[evt.Name]
	if ok {
		goWithWaitGroup(&wg, func() {
			handler.EventHook(ctx, evt)
		})
	}

	wg.Wait()
}

// Types for bridge events.
const (
	// BridgeMessageSent is event type for message sent event.
	BridgeMessageSent = BridgeEventType(MessageSent)

	// BridgeUserLeft is event type fired when user's leaving chat.
	BridgeUserJoin = BridgeEventType("user-join")

	// BridgeUserJoin is event type fired when user's joining chat.
	BridgeUserLeft = BridgeEventType("user-left")
)

type messageSubscriber struct {
	id        string
	requestID string
}

// BridgeMessageHandler handles sending, subscribing and
// receiving of message-sent type events.
type BridgeMessageHandler struct {
	bridge *Bridge
	log    *logrus.Logger

	channels map[messageSubscriber]chan<- sse.Event
	mtx      *sync.RWMutex
}

// NewBridgeMessageHandler is default and safe constructor for
// BridgeMessageHandler.
func NewBridgeMessageHandler(log *logrus.Logger) *BridgeMessageHandler {
	return &BridgeMessageHandler{
		log:      log,
		channels: make(map[messageSubscriber]chan<- sse.Event),
		mtx:      &sync.RWMutex{},
	}
}

// Subscribe given ID for SSE events. Returns unsubscribe func.
func (a *BridgeMessageHandler) Subscribe(ctx context.Context, req MessageSubscribeRequest) func() {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	key := messageSubscriber{
		id:        req.ID,
		requestID: req.RequestID,
	}

	log := a.log.WithFields(logrus.Fields{
		"reqID": req.RequestID,
		"subID": req.ID,
	})

	a.channels[key] = req.Channel
	log.Info("Client has subscribed for bridge message handler.")

	unsubscribe := func() {
		a.mtx.Lock()
		delete(a.channels, key)
		a.mtx.Unlock()
		log.Info("Client has unsubscribed from bridge message handler.")
	}
	return unsubscribe
}

// EventHook for SSE events sent to browsers.
func (a *BridgeMessageHandler) EventHook(_ context.Context, evt BridgeEvent) {
	a.mtx.RLock()
	defer a.mtx.RUnlock()

	if evt.Headers.Get(bridgeContentTypeHeaderVar) != contentTypeApplicationJSON {
		a.log.WithFields(logrus.Fields{
			"eventType": string(evt.Name),
			"eventID":   evt.ID,
			"reqID":     evt.Headers.Get(bridgeRequestIDHeaderVar),
			"scope":     "BridgeMessageHandler.EventHook",
		}).Error("Invalid content type of event data.")
		return
	}

	for _, c := range a.channels {
		c <- sse.Event{
			ID:   evt.ID,
			Type: string(evt.Name),
			Data: evt.Data,
		}
	}
}

const (
	bridgeRequestIDHeaderVar   = "Request-ID"
	bridgeContentTypeHeaderVar = "Content-Type"
	contentTypeApplicationJSON = "application/json; charset=utf-8"
)

// BridgeEventProducer publishes events with given T type to event bridge.
type BridgeEventProducer[T any] struct {
	EventBridge *Bridge
	Type        BridgeEventType
	Log         *logrus.Logger
	Clock
}

// SendEvent publishes event with given data of T type and unique ID.
func (p *BridgeEventProducer[T]) SendEvent(ctx context.Context, id string, evt T) {
	data, err := json.Marshal(evt)
	if err != nil {
		p.Log.WithFields(logrus.Fields{
			"eventID": id,
			"reqID":   middleware.GetReqID(ctx),
			"scope":   "BridgeEventProducer.SendEvent",
		}).Error("Failed to encode data as json.")
		return
	}

	p.EventBridge.SendEvent(BridgeEvent{
		ID:        id,
		Name:      p.Type,
		CreatedAt: p.Now().UnixMicro(),
		Headers: BridgeHeaders{
			bridgeContentTypeHeaderVar: "application/json; charset=utf-8",
			bridgeRequestIDHeaderVar:   middleware.GetReqID(ctx),
		},
		Data: data,
	})
}
