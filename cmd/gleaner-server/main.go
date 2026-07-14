package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soulteary/gleaner"
)

func main() {
	host := envOr("GLEANER_HOST", "0.0.0.0")
	port := envOr("GLEANER_PORT", "3824")
	if err := run(context.Background(), host, port, os.Stderr, signal.Notify); err != nil {
		log.Fatal(err)
	}
}

type signalNotifier func(c chan<- os.Signal, sig ...os.Signal)

func run(ctx context.Context, host, port string, logOut io.Writer, notify signalNotifier) error {
	logger := log.New(logOut, "", log.LstdFlags)

	client, err := gleaner.NewClient(gleaner.OptionsFromEnv())
	if err != nil {
		return fmt.Errorf("init gleaner: %w", err)
	}
	defer client.Close()

	ln, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler: newHandler(client),
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("Server running at http://%s/ (healthz at /healthz)", ln.Addr().String())
		errCh <- srv.Serve(ln)
	}()

	stop := make(chan os.Signal, 1)
	notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-stop:
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown: %v", err)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
