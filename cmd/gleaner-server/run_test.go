package main

import (
	"bytes"
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

func TestRunStartsAndShutsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var logBuf bytes.Buffer
	notify := func(c chan<- os.Signal, _ ...os.Signal) {}

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, "127.0.0.1", "0", &logBuf, notify)
	}()

	var addr string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
		line := logBuf.String()
		if strings.Contains(line, "Server running at http://") {
			start := strings.Index(line, "http://")
			end := strings.Index(line[start:], "/ (healthz")
			if start >= 0 && end > 0 {
				addr = line[start : start+end]
				break
			}
		}
	}
	if addr == "" {
		t.Fatalf("server did not start, log=%q", logBuf.String())
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
	case <-time.After(5 * time.Second):
		t.Fatal("run did not exit after cancel")
	}
}

func TestRunWithSignal(t *testing.T) {
	ctx := context.Background()
	var logBuf io.Writer = &bytes.Buffer{}
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
	case <-time.After(5 * time.Second):
		t.Fatal("run did not exit on signal")
	}
}
