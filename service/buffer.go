package service

import (
	"context"
	"sync"
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
