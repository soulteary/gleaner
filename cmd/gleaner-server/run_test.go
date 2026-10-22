package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("GLEANER_TEST_KEY", "")
	if envOr("GLEANER_TEST_KEY", "fallback") != "fallback" {
		t.Fatal("expected fallback")
	}
	t.Setenv("GLEANER_TEST_KEY", "custom")
	if envOr("GLEANER_TEST_KEY", "fallback") != "custom" {
		t.Fatal("expected custom value")
	}
}

// startupLog observes the complete startup log without racing a bytes.Buffer.
type startupLog struct {
	ready chan string
}

func (l startupLog) Write(p []byte) (int, error) {
	line := string(p)
	if start := strings.Index(line, "http://"); start >= 0 {
		if end := strings.Index(line[start:], "/ (healthz"); end > 0 {
			select {
			case l.ready <- line[start : start+end]:
			default:
			}
		}
	}
	return len(p), nil
}

func TestRunStartsAndShutsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan string, 1)
	logOut := startupLog{ready: ready}
	notify := func(c chan<- os.Signal, _ ...os.Signal) {}

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, "127.0.0.1", "0", logOut, notify)
	}()

	var addr string
	// Loading a real dictionary can exceed three seconds on a cold CI runner.
	select {
	case addr = <-ready:
	case err := <-errCh:
		t.Fatalf("server exited before startup: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("server did not start")
	}

	resp, err := http.Get(addr + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("run did not exit after cancel")
	}
}

func TestRunWithSignal(t *testing.T) {
	ctx := context.Background()
	var logBuf io.Writer = io.Discard
	notify := func(c chan<- os.Signal, _ ...os.Signal) {
		go func() {
			time.Sleep(100 * time.Millisecond)
			c <- os.Interrupt
		}()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, "127.0.0.1", "0", logBuf, notify)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("run did not exit on signal")
	}
}
