package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/fenole/szmaterlok/service"
)

func run(ctx context.Context) error {
	log := service.LoggerDefault()
	log.SetLevel(logrus.DebugLevel)

	if err := service.ConfigLoad(context.TODO()); err != nil {
		return err
	}

	config := service.ConfigDefault()
	if err := service.ConfigRead(&config); err != nil {
		return err
	}

	tokenizer, err := service.NewSessionAgeTokenizer(config.SessionSecret)
	if err != nil {
		return err
	}

	tokenizerCache := service.NewSessionTokenizerCache(service.SessionTokenizerCacheBuilder{
		Wrapped: tokenizer,
		Timeout: time.Second * 5,
		Logger:  log,
	})

	bridge := service.NewBridge(ctx)
	messageHandler := service.NewBridgeMessageHandler(bridge, log)

	bridge.Hook(service.BridgeMessageSent, messageHandler)

	r := service.NewRouter(service.RouterDependencies{
		Logger: log,
		SessionStore: &service.SessionCookieStore{
			ExpirationTime: time.Hour * 24 * 7,
			Tokenizer:      tokenizerCache,
			Clock:          service.ClockFunc(time.Now),
		},
		MessageSender:   messageHandler,
		MessageNotifier: messageHandler,
		IDGenerator:     service.IDGeneratorFunc(uuid.NewString),
		Clock:           service.ClockFunc(time.Now),
	})

	c := make(chan os.Signal, 1)
	errc := make(chan error, 1)

	wait := time.Second * 15
	srv := &http.Server{
		Addr:    config.Address,
		Handler: r,
		// TODO(thinkofher): Come back later to setup timeouts.
	}

	log.Println("Starting szmaterlok")
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Printf("Listening at %s", config.Address)
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
		ctx, cancel := context.WithTimeout(ctx, wait)
		defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.
		srv.Shutdown(ctx)

		// Wait for bridge to process its jobs.
		bridge.Shutdown(ctx)

		// Optionally, you could run srv.Shutdown in a goroutine and block on
		// <-ctx.Done() if your application should wait for other services
		// to finalize based on context cancellation.
		log.Println("Shutting down")
		return nil
	case err := <-errc:
		return err
	}
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal("szmaterlok:", err.Error())
	}
}
