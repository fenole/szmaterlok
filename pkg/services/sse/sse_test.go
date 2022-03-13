package sse

import (
	"testing"

	"github.com/matryer/is"
)

func TestEventStream(t *testing.T) {
	type testArgs struct {
		name  string
		event Event
		want  string
	}

	scenario := func(tt testArgs) (string, func(*testing.T)) {
		return tt.name, func(t *testing.T) {
			is := is.New(t)

			stream, err := tt.event.Stream()
			is.NoErr(err)
			is.True(stream != nil)

			is.Equal(string(stream), tt.want)
		}
	}

	t.Run(scenario(testArgs{
		name: "minimal event",
		event: Event{
			Type: "usermessage",
			Data: []byte(`{"username": "bobby", "time": "02:34:11", "text": "Hi everyone."}`),
		},
		want: `event: usermessage
data: {"username": "bobby", "time": "02:34:11", "text": "Hi everyone."}

`,
	}))

	t.Run(scenario(testArgs{
		name: "event with id",
		event: Event{
			Type: "notifyusers",
			Data: []byte(`{"which": "all", "time": "2:34:11", "text": "This is notification."}`),
			ID:   "someid",
		},
		want: `event: notifyusers
id: someid
data: {"which": "all", "time": "2:34:11", "text": "This is notification."}

`,
	}))

	t.Run(scenario(testArgs{
		name: "multiline data with id and retry value",
		event: Event{
			Type:  "hugevent",
			Data:  []byte("one\ntwo\nthree"),
			ID:    "someotherid",
			Retry: 2137,
		},
		want: `event: hugevent
id: someotherid
retry: 2137
data: one
data: two
data: three

`,
	}))
}
