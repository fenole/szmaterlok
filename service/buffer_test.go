package service

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/matryer/is"
)

func TestMessageCircularBuffer(t *testing.T) {
	t.Run("PushEvent", func(t *testing.T) {
		t.Run("Single item", func(t *testing.T) {
			ctx := context.TODO()
			is := is.New(t)

			want := EventSentMessage{
				ID: "someid",
			}
			b := NewMessageCircularBuffer(1)
			b.PushEvent(ctx, want)

			got := *b.head.value

			is.Equal(got, want)
		})
		t.Run("Two items", func(t *testing.T) {
			ctx := context.TODO()
			is := is.New(t)

			first := EventSentMessage{
				ID: "first",
			}
			second := EventSentMessage{
				ID: "second",
			}

			b := NewMessageCircularBuffer(2)
			b.PushEvent(ctx, first)
			b.PushEvent(ctx, second)

			is.Equal(first, *b.head.value)
			is.Equal(second, *b.head.next.value)
		})
		t.Run("overwrite", func(t *testing.T) {
			ctx := context.TODO()
			is := is.New(t)

			first := EventSentMessage{
				ID: "first",
			}
			second := EventSentMessage{
				ID: "second",
			}

			b := NewMessageCircularBuffer(2)
			b.PushEvent(ctx, EventSentMessage{})
			b.PushEvent(ctx, EventSentMessage{})
			b.PushEvent(ctx, EventSentMessage{})
			b.PushEvent(ctx, EventSentMessage{})
			b.PushEvent(ctx, EventSentMessage{})
			b.PushEvent(ctx, first)
			b.PushEvent(ctx, second)

			is.Equal(first, *b.head.value)
			is.Equal(second, *b.head.next.value)

		})
	})
	t.Run("BufferedEvents", func(t *testing.T) {
		t.Run("sync", func(t *testing.T) {
			scenario := func(size func([]EventSentMessage) int) func(*testing.T) {
				return func(t *testing.T) {
					ctx := context.TODO()

					is := is.New(t)

					events := []EventSentMessage{
						{
							ID: "1",
						},
						{
							ID: "2",
						},
						{
							ID: "3",
						},
					}

					b := NewMessageCircularBuffer(size(events))

					for _, e := range events {
						b.PushEvent(ctx, e)
					}

					got := b.BufferedEvents(ctx)
					sort.Slice(got, func(i, j int) bool {
						return got[i].ID < got[j].ID
					})

					is.Equal(len(got), len(events))
					is.Equal(got, events)
				}
			}

			t.Run("full", scenario(func(e []EventSentMessage) int {
				return len(e)
			}))
			t.Run("with space left", scenario(func(e []EventSentMessage) int {
				return len(e) + 10
			}))
		})
		t.Run("concurrent", func(t *testing.T) {
			ctx := context.TODO()

			is := is.New(t)

			events := []EventSentMessage{
				{
					ID: "1",
				},
				{
					ID: "2",
				},
				{
					ID: "3",
				},
				{
					ID: "4",
				},
			}

			b := NewMessageCircularBuffer(len(events))

			wg := &sync.WaitGroup{}
			wg.Add(len(events))
			for _, e := range events {
				e := e
				go func() {
					defer wg.Done()
					b.PushEvent(ctx, e)
				}()
			}
			wg.Wait()

			workers := 10
			got := make([][]EventSentMessage, workers, workers)

			wg.Add(workers)
			for i := 0; i < workers; i++ {
				i := i
				go func() {
					defer wg.Done()
					got[i] = b.BufferedEvents(ctx)
				}()
			}
			wg.Wait()

			for _, g := range got {
				sort.Slice(g, func(i, j int) bool {
					return g[i].ID < g[j].ID
				})

				is.Equal(len(g), len(events))
				is.Equal(g, events)
			}

		})
	})
}
