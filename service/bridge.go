package service

import (
	"context"
	"sync"
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

	hooks map[BridgeEventType]bridgeEventHandlerComposite
}

// NewBridge is constructor for event bridge. It returns
// default instance of event bridge.
func NewBridge(ctx context.Context) *Bridge {
	evtChan := make(chan BridgeEvent)
	res := &Bridge{
		queue:  evtChan,
		closer: make(chan struct{}),
		hooks: map[BridgeEventType]bridgeEventHandlerComposite{
			BridgeEventGlob: {},
		},
	}

	go res.run(ctx)
	return res
}

// SendEvent sends event to event bridge. It blocks, so it's
// a good idea to run it in a separate goroutine.
func (b *Bridge) SendEvent(evt BridgeEvent) {
	b.queue <- evt
}

// Hook adds given event handler to hook list for given event type.
// Given hook will be fired as soon as event bridge receives new event
// with matching event type.
//
// All hooks should be added before sending events to event bridge.
func (b *Bridge) Hook(t BridgeEventType, h BridgeEventHandler) {
	_, ok := b.hooks[t]
	if !ok {
		b.hooks[t] = bridgeEventHandlerComposite{}
	}

	b.hooks[t] = append(b.hooks[t], h)
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

// run is main event loop of event bridge.
func (b *Bridge) run(ctx context.Context) {
	wg := sync.WaitGroup{}

	// Helper for running jobs with the help
	// of wait group for further synchronization.
	wgGo := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}

	// Main processing loop.
	for evt := range b.queue {
		evt := evt

		globHandler, ok := b.hooks[BridgeEventGlob]
		if ok {
			wgGo(func() {
				globHandler.EventHook(ctx, evt)
			})
		}

		handler, ok := b.hooks[evt.Name]
		if ok {
			wgGo(func() {
				handler.EventHook(ctx, evt)
			})
		}
	}

	// Wait for all jobs to finish.
	wg.Wait()

	// Send signal to closer and indicate event loop has finished.
	b.closer <- struct{}{}
}
