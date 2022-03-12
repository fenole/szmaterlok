package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/fenole/szmaterlok/web"
)

func run() error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

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