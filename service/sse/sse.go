// Package sse implements Server Sent Event API for szmaterlok
// backend server. You'll find here both type models and
// http methods and functions.
package sse

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// Event is a simple stream of text data which must be encoded using UTF-8.
// Messages in the event stream are separated by a pair of newline characters. A colon
// as the first character of a line is in essence a comment, and is ignored.
//
// Sample event stream.
//
//		event: usermessage
//		data: {"username": "bobby", "time": "02:34:11", "text": "Hi everyone."}
//
// Below and above documentation is mostly copied from MDN.
// Source: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
type Event struct {
	// Type is a string identifying the type of event described. If
	// this is specified, an event will be dispatched on the browser
	// to the listener for the specified event name; the website source
	// code should use addEventListener() to listen for named events.
	Type string

	// Data is the field for the message.
	Data []byte

	// ID is unique event identifier.
	ID string

	// Retry is reconnection time. If the connection to the server is lost, the
	// browser will wait for the specified time before attempting to reconnect.
	// This must be an integer, specifying the reconnection time in milliseconds.
	// If a non-integer value is specified, the field is ignored.
	Retry int64
}

// Stream encodes Event into text/event-stream format and
// returns it as slice of bytes.
func (e Event) Stream() ([]byte, error) {
	buff := &bytes.Buffer{}

	if err := Encode(buff, e); err != nil {
		return nil, fmt.Errorf("sse.Encode: %w", err)
	}
	return buff.Bytes(), nil
}

// Encode writes the event stream encoding of v to the stream,
// followed by a newline character.
func Encode(stream io.Writer, v Event) error {
	if _, err := fmt.Fprintf(stream, "event: %s\n", v.Type); err != nil {
		return fmt.Errorf("fmt.Fprintf: %w", err)
	}

	if v.ID != "" {
		if _, err := fmt.Fprintf(stream, "id: %s\n", v.ID); err != nil {
			return fmt.Errorf("fmt.Fprintf: %w", err)
		}
	}

	if v.Retry != 0 {
		if _, err := fmt.Fprintf(stream, "retry: %d\n", v.Retry); err != nil {
			return fmt.Errorf("fmt.Fprintf: %w", err)
		}
	}

	for _, l := range bytes.Split(v.Data, []byte("\n")) {
		if _, err := fmt.Fprintf(stream, "data: %s\n", l); err != nil {
			return fmt.Errorf("fmt.Fprintf: %w", err)
		}
	}
	if _, err := fmt.Fprint(stream, "\n"); err != nil {
		return fmt.Errorf("fmt.Fprintf: %w", err)
	}

	return nil
}

// ContentTypeEventStream is content type for event stream filetype.
const ContentTypeEventStream string = "text/event-stream"

// Headers is middleware that sets up http headers for SSE http handler.
func Headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeEventStream)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		next.ServeHTTP(w, r)
	})
}
