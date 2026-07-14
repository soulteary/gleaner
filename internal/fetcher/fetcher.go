package fetcher

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	defaultUserAgent    = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/60.0.3112.113 Chrome/60.0.3112.113 Safari/537.36"
	DefaultMaxBodyBytes = 10 << 20 // 10 MiB
)

var (
	charsetMetaRe   = regexp.MustCompile(`(?i)<meta[^>]+charset\s*=\s*"?([\w\d-]+)`)
	ErrBodyTooLarge = errors.New("response body exceeds limit")
)

// Client fetches web pages with encoding detection.
type Client struct {
	HTTPClient       *http.Client
	MaxBodyBytes     int64
	BlockPrivateURLs bool
}

// NewClient creates a fetcher with sensible defaults.
func NewClient() *Client {
	c := &Client{
		MaxBodyBytes:     DefaultMaxBodyBytes,
		BlockPrivateURLs: true,
	}
	c.HTTPClient = c.buildHTTPClient()
	return c
}

func (c *Client) buildHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if c.BlockPrivateURLs {
				if err := checkDialAddr(ctx, addr); err != nil {
					return nil, err
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if c.BlockPrivateURLs {
				if err := validateFetchURL(req.URL.String()); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// Get fetches a URL and returns decoded HTML text.
func (c *Client) Get(url string) (string, error) {
	return c.GetContext(context.Background(), url)
}

// GetContext fetches a URL with cancellation and deadline support.
func (c *Client) GetContext(ctx context.Context, url string) (string, error) {
	if c.BlockPrivateURLs {
		if err := validateFetchURL(url); err != nil {
			return "", err
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,en-US;q=0.6,en;q=0.4")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := c.readBody(resp)
	if err != nil {
		return "", err
	}

	encoding := detectEncoding(resp.Header.Get("Content-Type"), body)
	return decodeBody(body, encoding)
}

func (c *Client) maxBodyBytes() int64 {
	if c.MaxBodyBytes > 0 {
		return c.MaxBodyBytes
	}
	return DefaultMaxBodyBytes
}

func (c *Client) readBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	enc := resp.Header.Get("Content-Encoding")
	switch strings.ToLower(enc) {
	case "gzip":
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		reader = gr
	case "deflate":
		zr, err := zlib.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		reader = zr
	}
	limited := io.LimitReader(reader, c.maxBodyBytes()+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > c.maxBodyBytes() {
		return nil, fmt.Errorf("%w (%d bytes)", ErrBodyTooLarge, c.maxBodyBytes())
	}
	return body, nil
}

func detectEncoding(contentType string, body []byte) string {
	if contentType != "" {
		ct := strings.ToLower(contentType)
		if idx := strings.Index(ct, "charset="); idx >= 0 {
			cs := strings.TrimSpace(strings.Trim(ct[idx+8:], `"' `))
			return normalizeEncoding(cs)
		}
	}
	if m := charsetMetaRe.FindSubmatch(body); len(m) > 1 {
		return normalizeEncoding(string(m[1]))
	}
	return "utf-8"
}

func normalizeEncoding(enc string) string {
	enc = strings.ToLower(strings.ReplaceAll(enc, "-", ""))
	switch enc {
	case "gb2312", "gbk", "gb18030":
		return "gbk"
	default:
		return enc
	}
}

func decodeBody(body []byte, encoding string) (string, error) {
	switch encoding {
	case "gbk", "gb2312", "gb18030":
		r := transform.NewReader(bytes.NewReader(body), simplifiedchinese.GBK.NewDecoder())
		decoded, err := io.ReadAll(r)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	default:
		return string(body), nil
	}
}

// DetectEncoding exposes encoding detection for tests.
func DetectEncoding(contentType string, body []byte) string {
	return detectEncoding(contentType, body)
}
