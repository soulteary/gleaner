package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/gleaner"
)

func TestHealthz(t *testing.T) {
	client := testClient(t)
	handler := newHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestServerInvalidJSON(t *testing.T) {
	client := testClient(t)
	handler := newHandler(client)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "error") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestServerMissingInput(t *testing.T) {
	client := testClient(t)
	handler := newHandler(client)
	body := bytes.NewBufferString(`{"size":100}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServerHTMLInput(t *testing.T) {
	client := testClient(t)
	handler := newHandler(client)
	payload := `{"html":"<html><head><title>T</title></head><body><p>正文测试内容足够长。</p></body></html>","size":120}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var article gleaner.Article
	if err := json.Unmarshal(rec.Body.Bytes(), &article); err != nil {
		t.Fatal(err)
	}
	if article.Title == "" || article.Content == "" {
		t.Fatalf("article=%+v", article)
	}
}

func TestServerFetchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer srv.Close()

	client := testClientWithOptions(t, gleaner.Options{BlockPrivateURLs: false})
	handler := newHandler(client)
	payload := `{"url":"` + srv.URL + `","size":120}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "http status") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestServerFetchBodyTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 4096))
	}))
	defer srv.Close()

	opts := gleaner.DefaultOptions()
	opts.BlockPrivateURLs = false
	opts.FetcherMaxBodyBytes = 1024
	client := testClientWithOptions(t, opts)
	handler := newHandler(client)
	payload := `{"url":"` + srv.URL + `","size":120}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "exceeds limit") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestServerFetchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = io.WriteString(w, "<html><body>late</body></html>")
	}))
	defer srv.Close()

	opts := gleaner.DefaultOptions()
	opts.BlockPrivateURLs = false
	opts.FetcherHTTPTimeout = 50 * time.Millisecond
	client := testClientWithOptions(t, opts)
	handler := newHandler(client)
	payload := `{"url":"` + srv.URL + `","size":120}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServerBlockedPrivateURL(t *testing.T) {
	client := testClientWithOptions(t, gleaner.Options{BlockPrivateURLs: true})
	handler := newHandler(client)
	payload := `{"url":"http://127.0.0.1:3824/","size":120}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "blocked") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestServerMethodNotAllowed(t *testing.T) {
	client := testClient(t)
	handler := newHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rec.Code)
	}
}

func testClient(t *testing.T) *gleaner.Client {
	return testClientWithOptions(t, gleaner.DefaultOptions())
}

func testClientWithOptions(t *testing.T, opts gleaner.Options) *gleaner.Client {
	t.Helper()
	client, err := gleaner.NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}
