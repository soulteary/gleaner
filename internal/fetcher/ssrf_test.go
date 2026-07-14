package fetcher_test

import (
	"context"
	"strings"
	"testing"

	"github.com/soulteary/gleaner/internal/fetcher"
)

func TestSSRFBlockedHostname(t *testing.T) {
	c := fetcher.NewClient()
	_, err := c.Get("http://localhost/secret")
	if err == nil {
		t.Fatal("expected blocked url error")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("err=%v", err)
	}
}

func TestSSRFBlockedPrivateIP(t *testing.T) {
	c := fetcher.NewClient()
	_, err := c.Get("http://192.168.1.1/internal")
	if err == nil {
		t.Fatal("expected blocked url error")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("err=%v", err)
	}
}

func TestSSRFBlockedLoopbackIP(t *testing.T) {
	c := fetcher.NewClient()
	_, err := c.Get("http://127.0.0.1/admin")
	if err == nil {
		t.Fatal("expected blocked url error")
	}
}

func TestSSRFBlockedScheme(t *testing.T) {
	c := fetcher.NewClient()
	_, err := c.Get("file:///etc/passwd")
	if err == nil {
		t.Fatal("expected scheme error")
	}
}

func TestSSRFAllowedPublicHost(t *testing.T) {
	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetContext(ctx, "http://example.com/")
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
	if strings.Contains(err.Error(), "blocked") {
		t.Fatalf("public host should not be blocked: %v", err)
	}
}
