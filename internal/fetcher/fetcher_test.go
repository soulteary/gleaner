package fetcher_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"github.com/soulteary/gleaner/internal/fetcher"
)

func TestGetUTF8(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>hello</body></html>"))
	}))
	defer srv.Close()

	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	body, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "hello") {
		t.Fatalf("body=%q", body)
	}
}

func TestGetGBK(t *testing.T) {
	plain := "<html><body>中文页面</body></html>"
	var buf bytes.Buffer
	w := transform.NewWriter(&buf, simplifiedchinese.GBK.NewEncoder())
	_, _ = w.Write([]byte(plain))
	_ = w.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=gbk")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	body, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "中文页面") {
		t.Fatalf("body=%q", body)
	}
}

func TestGetGzip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte("<html>gzip page</html>"))
		_ = gz.Close()
	}))
	defer srv.Close()

	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	body, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "gzip page") {
		t.Fatalf("body=%q", body)
	}
}

func TestRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>final</html>"))
	}))
	defer final.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirect.Close()

	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	body, err := c.Get(redirect.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "final") {
		t.Fatalf("body=%q", body)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = io.WriteString(w, "slow")
	}))
	defer srv.Close()

	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	c.HTTPClient.Timeout = 50 * time.Millisecond
	_, err := c.Get(srv.URL)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestGetContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("<html>late</html>"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	_, err := c.GetContext(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestMaxBodyBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 2048))
	}))
	defer srv.Close()

	c := fetcher.NewClient()
	c.BlockPrivateURLs = false
	c.MaxBodyBytes = 1024
	_, err := c.Get(srv.URL)
	if err == nil {
		t.Fatal("expected body too large error")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("err=%v", err)
	}
}

func TestDetectEncodingMeta(t *testing.T) {
	body := []byte(`<html><head><meta charset="gbk"></head></html>`)
	enc := fetcher.DetectEncoding("text/html", body)
	if enc != "gbk" {
		t.Fatalf("encoding=%q", enc)
	}
}
