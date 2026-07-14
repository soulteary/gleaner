package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testHTML = `<!DOCTYPE html><html><head><title>CLI测试</title></head><body>
<p>这是用于命令行测试的正文内容，需要足够长以便生成摘要。</p>
<p>第二段正文继续补充信息，使提取器能够识别文章主体区域。</p>
</body></html>`

func TestRunMissingInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRunInvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-not-a-flag"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
}

func TestRunHTMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "article.html")
	if err := os.WriteFile(path, []byte(testHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-html", path, "-size", "80"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "title: CLI测试") || !strings.Contains(out, "summary:") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestRunHTMLPositional(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "article.html")
	if err := os.WriteFile(path, []byte(testHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{path, "80"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "content:") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestRunJSONOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "article.html")
	if err := os.WriteFile(path, []byte(testHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-html", path, "-json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"title"`) {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestRunMissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-html", "/nonexistent/path.html"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected error on stderr")
	}
}

func TestRunURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(testHTML))
	}))
	defer srv.Close()

	t.Setenv("GLEANER_BLOCK_PRIVATE_URLS", "false")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-url", srv.URL, "-size", "80"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "title: CLI测试") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestRunURLPositional(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(testHTML))
	}))
	defer srv.Close()

	t.Setenv("GLEANER_BLOCK_PRIVATE_URLS", "false")

	var stdout, stderr bytes.Buffer
	code := run([]string{srv.URL, "80"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "summary:") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestRunShingle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "article.html")
	if err := os.WriteFile(path, []byte(testHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-html", path, "-shingle", "-size", "80"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}
