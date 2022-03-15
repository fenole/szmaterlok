package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/fenole/szmaterlok/service/sse"
	"github.com/fenole/szmaterlok/web"
)

func run() error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
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
	r.Handle("/*", http.FileServer(http.FS(web.Assets)))

	c := make(chan os.Signal, 1)
	errc := make(chan error, 1)

	wait := time.Second * 15
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
		// TODO(thinkofher): Come back later to setup timeouts.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Println("Listening at 0.0.0.0:8080...")
		if err := srv.ListenAndServe(); err != nil {
			errc <- err
		}
	}()

	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal or error from server.
	select {
	case <-c:
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.
		srv.Shutdown(ctx)
		// Optionally, you could run srv.Shutdown in a goroutine and block on
		// <-ctx.Done() if your application should wait for other services
		// to finalize based on context cancellation.
		log.Println("shutting down")
		return nil
	case err := <-errc:
		return err
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal("szmaterlok:", err.Error())
	}
}
