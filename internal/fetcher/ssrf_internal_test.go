package fetcher

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	if !isBlockedIP(nil) {
		t.Fatal("nil IP should be blocked")
	}
	if !isBlockedIP(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback should be blocked")
	}
	if !isBlockedIP(net.ParseIP("192.168.1.1")) {
		t.Fatal("private should be blocked")
	}
	if !isBlockedIP(net.ParseIP("169.254.1.1")) {
		t.Fatal("link-local should be blocked")
	}
	if !isBlockedIP(net.ParseIP("fc00::1")) {
		t.Fatal("IPv6 ULA should be blocked")
	}
	if isBlockedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public IP should not be blocked")
	}
}

func TestValidateFetchURL(t *testing.T) {
	if err := validateFetchURL("not-a-url"); err == nil {
		t.Fatal("expected parse error")
	}
	if err := validateFetchURL("file:///etc/passwd"); err == nil {
		t.Fatal("expected scheme error")
	}
	if err := validateFetchURL("http:///path"); err == nil {
		t.Fatal("expected missing host error")
	}
	if err := validateFetchURL("http://localhost/"); err == nil {
		t.Fatal("expected blocked hostname")
	}
	if err := validateFetchURL("http://192.168.0.1/"); err == nil {
		t.Fatal("expected blocked IP")
	}
	if err := validateFetchURL("http://example.com/"); err != nil {
		t.Fatalf("public host should pass: %v", err)
	}
}

func TestCheckDialAddrBlockedHostname(t *testing.T) {
	err := checkDialAddr(context.Background(), "localhost:80")
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("err=%v", err)
	}
}

func TestCheckDialAddrBlockedIP(t *testing.T) {
	err := checkDialAddr(context.Background(), "127.0.0.1:8080")
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("err=%v", err)
	}
}

func TestCheckDialAddrPublicIP(t *testing.T) {
	err := checkDialAddr(context.Background(), "8.8.8.8:443")
	if err != nil {
		t.Fatalf("public IP should pass: %v", err)
	}
}

func TestCheckDialAddrHostWithoutPort(t *testing.T) {
	err := checkDialAddr(context.Background(), "127.0.0.1")
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("err=%v", err)
	}
}

func TestIsBlockedHostname(t *testing.T) {
	if !isBlockedHostname("localhost") {
		t.Fatal("localhost blocked")
	}
	if !isBlockedHostname("app.localhost") {
		t.Fatal(".localhost suffix blocked")
	}
	if !isBlockedHostname("metadata.google.internal") {
		t.Fatal("metadata host blocked")
	}
	if isBlockedHostname("example.com") {
		t.Fatal("public host not blocked")
	}
}

func TestBuildHTTPClientDialBlocked(t *testing.T) {
	c := NewClient()
	tr, ok := c.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	conn, err := tr.DialContext(context.Background(), "tcp", "127.0.0.1:80")
	if conn != nil {
		conn.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("dial err=%v", err)
	}
}

func TestReadBodyDeflate(t *testing.T) {
	c := NewClient()
	c.MaxBodyBytes = 1024
	// deflate path is covered via readBody when Content-Encoding is deflate
	resp := &http.Response{
		Header: http.Header{"Content-Encoding": []string{"deflate"}},
		Body:   http.NoBody,
	}
	_, err := c.readBody(resp)
	if err == nil {
		t.Fatal("expected error for empty deflate body")
	}
}

func TestMaxBodyBytesZeroUsesDefault(t *testing.T) {
	c := &Client{MaxBodyBytes: 0}
	if c.maxBodyBytes() != DefaultMaxBodyBytes {
		t.Fatalf("got=%d", c.maxBodyBytes())
	}
}
