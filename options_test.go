package gleaner

import (
	"os"
	"testing"
	"time"

	"github.com/soulteary/gleaner/internal/segment"
)

func init() {
	_ = os.Setenv("GLEANER_BLOCK_PRIVATE_URLS", "false")
}

func TestEnvBool(t *testing.T) {
	t.Setenv("GLEANER_TEST_BOOL", "")
	if !envBool("GLEANER_TEST_BOOL", true) {
		t.Fatal("empty should use fallback true")
	}
	if envBool("GLEANER_TEST_BOOL", false) {
		t.Fatal("empty should use fallback false")
	}

	for _, v := range []string{"1", "true", "yes", "on", "TRUE"} {
		t.Setenv("GLEANER_TEST_BOOL", v)
		if !envBool("GLEANER_TEST_BOOL", false) {
			t.Fatalf("expected true for %q", v)
		}
	}
	for _, v := range []string{"0", "false", "no", "off"} {
		t.Setenv("GLEANER_TEST_BOOL", v)
		if envBool("GLEANER_TEST_BOOL", true) {
			t.Fatalf("expected false for %q", v)
		}
	}
	t.Setenv("GLEANER_TEST_BOOL", "maybe")
	if !envBool("GLEANER_TEST_BOOL", true) {
		t.Fatal("invalid value should use fallback")
	}
}

func TestOptionsFromEnv(t *testing.T) {
	t.Setenv("GLEANER_SEGMENT_BACKEND", "gse")
	t.Setenv("GLEANER_BLOCK_PRIVATE_URLS", "false")
	opts := OptionsFromEnv()
	if opts.SegmentBackend != segment.BackendGSE {
		t.Fatalf("backend=%q", opts.SegmentBackend)
	}
	if opts.BlockPrivateURLs {
		t.Fatal("expected BlockPrivateURLs false")
	}

	t.Setenv("GLEANER_SEGMENT_BACKEND", "gojieba")
	opts = OptionsFromEnv()
	if opts.SegmentBackend != segment.BackendGojieba {
		t.Fatalf("backend=%q", opts.SegmentBackend)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.MinSummaryLen != 150 || opts.MaxSummaryLen != 350 {
		t.Fatalf("opts=%+v", opts)
	}
}

func TestDefaultExtractOptions(t *testing.T) {
	shingle := true
	opts := DefaultExtractOptions(200, &shingle)
	if opts.ExpectedLen != 200 || opts.Shingle == nil || !*opts.Shingle {
		t.Fatalf("opts=%+v", opts)
	}
}

func TestNewClientFetcherOptions(t *testing.T) {
	opts := DefaultOptions()
	opts.FetcherMaxBodyBytes = 2048
	opts.FetcherHTTPTimeout = 5 * time.Second
	client, err := NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.fetcher.MaxBodyBytes != 2048 {
		t.Fatalf("MaxBodyBytes=%d", client.fetcher.MaxBodyBytes)
	}
	if client.fetcher.HTTPClient.Timeout != 5*time.Second {
		t.Fatalf("timeout=%v", client.fetcher.HTTPClient.Timeout)
	}
}

func TestClientCloseNil(t *testing.T) {
	var c *Client
	c.Close()
}

func TestClientCloseDefaultClient(t *testing.T) {
	c, err := DefaultClient()
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}

func TestResolveExtractOpts(t *testing.T) {
	shingle := true
	client, err := NewClient(Options{Shingle: false, Filter: []string{"nav"}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	opts := ExtractOptions{
		Shingle:       &shingle,
		Filter:        []string{"ad"},
		MinSummaryLen: 10,
		MaxSummaryLen: 500,
	}
	gotShingle, filter, minLen, maxLen := client.resolveExtractOpts(opts)
	if !gotShingle || len(filter) != 1 || filter[0] != "ad" || minLen != 10 || maxLen != 500 {
		t.Fatalf("got shingle=%v filter=%v min=%d max=%d", gotShingle, filter, minLen, maxLen)
	}
}

func TestJoinLines(t *testing.T) {
	if joinLines(nil) != "" {
		t.Fatal("nil should be empty")
	}
	if joinLines([]string{"a", "b"}) != "a\nb" {
		t.Fatal("expected joined lines")
	}
}
