package gleaner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/soulteary/gleaner/internal/data"
	"github.com/soulteary/gleaner/internal/extractor"
	"github.com/soulteary/gleaner/internal/fetcher"
	"github.com/soulteary/gleaner/internal/nnmodel"
	"github.com/soulteary/gleaner/internal/segment"
	"github.com/soulteary/gleaner/internal/sentiment"
	"github.com/soulteary/gleaner/internal/summarizer"
)

// Article holds extracted article fields.
type Article struct {
	Content   string `json:"content"`
	Title     string `json:"title"`
	Pubdate   string `json:"pubdate"`
	Summary   string `json:"summary"`
	Sentiment int    `json:"sentiment"`
}

// Options configures Gleaner client defaults.
type Options struct {
	SegmentBackend      segment.Backend
	Shingle             bool
	MinSummaryLen       int
	MaxSummaryLen       int
	Filter              []string
	BlockPrivateURLs    bool
	FetcherMaxBodyBytes int64
	FetcherHTTPTimeout  time.Duration
}

// ExtractOptions per-call overrides for extraction and summarization.
type ExtractOptions struct {
	ExpectedLen   float64
	Shingle       *bool
	MinSummaryLen int
	MaxSummaryLen int
	Filter        []string
}

// DefaultExtractOptions returns extract options with expected summary length.
func DefaultExtractOptions(expectedLen float64, shingle *bool) ExtractOptions {
	return ExtractOptions{ExpectedLen: expectedLen, Shingle: shingle}
}

// DefaultOptions returns sensible defaults matching the legacy Node.js library.
func DefaultOptions() Options {
	return Options{
		SegmentBackend: segment.BackendGojieba,
		Shingle:        false,
		MinSummaryLen:  150,
		MaxSummaryLen:  350,
	}
}

// OptionsFromEnv builds Options with environment overrides.
func OptionsFromEnv() Options {
	opts := DefaultOptions()
	switch os.Getenv("GLEANER_SEGMENT_BACKEND") {
	case "gse":
		opts.SegmentBackend = segment.BackendGSE
	case "gojieba", "":
		opts.SegmentBackend = segment.BackendGojieba
	}
	opts.BlockPrivateURLs = envBool("GLEANER_BLOCK_PRIVATE_URLS", true)
	return opts
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// Client is the main Gleaner API client.
type Client struct {
	seg       segment.Segmenter
	model     *nnmodel.Model
	sentiment *sentiment.Analyzer
	fetcher   *fetcher.Client
	opts      Options
}

var (
	defaultClient *Client
	initOnce      sync.Once
	initErr       error
)

// NewClient creates a Gleaner client with options.
func NewClient(opts Options) (*Client, error) {
	seg, err := segment.New(opts.SegmentBackend, data.UserWordsTXT)
	if err != nil {
		seg, err = segment.Default(data.UserWordsTXT)
		if err != nil {
			return nil, fmt.Errorf("segmenter: %w", err)
		}
	}

	model, err := nnmodel.DefaultModel(data.SummaryJSON)
	if err != nil {
		return nil, fmt.Errorf("model: %w", err)
	}
	if err := model.Validate(); err != nil {
		return nil, err
	}

	sent, err := sentiment.New(seg, data.AFINNJSON, data.DUProtectionJSON)
	if err != nil {
		return nil, fmt.Errorf("sentiment: %w", err)
	}

	return &Client{
		seg:       seg,
		model:     model,
		sentiment: sent,
		fetcher:   newFetcher(opts),
		opts:      opts,
	}, nil
}

func newFetcher(opts Options) *fetcher.Client {
	fc := fetcher.NewClient()
	fc.BlockPrivateURLs = opts.BlockPrivateURLs
	if opts.FetcherMaxBodyBytes > 0 {
		fc.MaxBodyBytes = opts.FetcherMaxBodyBytes
	}
	if opts.FetcherHTTPTimeout > 0 {
		fc.HTTPClient.Timeout = opts.FetcherHTTPTimeout
	}
	return fc
}

// DefaultClient returns a lazily initialized default client.
// Do not call Close on the returned client; use a dedicated NewClient instead.
func DefaultClient() (*Client, error) {
	initOnce.Do(func() {
		defaultClient, initErr = NewClient(OptionsFromEnv())
	})
	return defaultClient, initErr
}

// Close releases segmenter resources. No-op for the package default singleton.
func (c *Client) Close() {
	if c == nil || c == defaultClient {
		return
	}
	if c.seg != nil {
		c.seg.Close()
	}
}

// GetArticleSync extracts article info from HTML synchronously.
func (c *Client) GetArticleSync(html string, expectedLen float64, shingle *bool) (*Article, error) {
	return c.GetArticleSyncWithOptions(html, DefaultExtractOptions(expectedLen, shingle))
}

// GetArticleSyncWithOptions extracts article with per-call options.
func (c *Client) GetArticleSyncWithOptions(html string, opts ExtractOptions) (*Article, error) {
	useShingle, filter, minLen, maxLen := c.resolveExtractOpts(opts)

	body := extractor.ExtractBody(html)
	body = extractor.FilterNoneContentTags(body)
	blocks := extractor.GetBlocks(body)
	blockScore := extractor.GetBlockScore(blocks, 5)
	groups := extractor.JudgeArticleBlock(blockScore)

	var articleBlocks []string
	if len(groups) > 0 {
		start := int(groups[0][0])
		end := int(groups[0][1])
		if start < len(blocks) && end <= len(blocks) {
			articleBlocks = blocks[start:end]
		}
	} else {
		articleBlocks = blocks
	}

	content := joinLines(articleBlocks)
	title := extractor.GetTitle(html)

	pubdate := ""
	if len(groups) > 0 {
		pubdate = extractor.GetPubdateBaseBlock(blocks, int(groups[0][0]))
	}
	if pubdate == "" {
		pubdate = extractor.GetPubdate(html)
	}

	if minLen == 0 {
		minLen = c.opts.MinSummaryLen
	}
	if maxLen == 0 {
		maxLen = c.opts.MaxSummaryLen
	}

	sum := summarizer.New(c.seg, c.model, useShingle, filter)
	expLen := summarizer.ResolveExpectedLength(articleBlocks, opts.ExpectedLen, minLen, maxLen)
	summary := sum.Summarize(content, expLen, title)

	return &Article{
		Content:   content,
		Title:     title,
		Pubdate:   pubdate,
		Summary:   summary,
		Sentiment: c.sentiment.Score(content),
	}, nil
}

// GetArticle fetches a URL and extracts article info.
func (c *Client) GetArticle(url string, expectedLen float64, shingle *bool) (*Article, error) {
	return c.GetArticleWithContext(context.Background(), url, DefaultExtractOptions(expectedLen, shingle))
}

// GetArticleWithContext fetches a URL with context and per-call options.
func (c *Client) GetArticleWithContext(ctx context.Context, url string, opts ExtractOptions) (*Article, error) {
	html, err := c.fetcher.GetContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return c.GetArticleSyncWithOptions(html, opts)
}

// Summarize generates a summary from HTML or text content.
func (c *Client) Summarize(content string, expectedLen float64, shingle *bool, title string) string {
	return c.SummarizeWithOptions(content, DefaultExtractOptions(expectedLen, shingle), title)
}

// SummarizeWithOptions generates a summary with per-call options.
func (c *Client) SummarizeWithOptions(content string, opts ExtractOptions, title string) string {
	useShingle, filter, minLen, maxLen := c.resolveExtractOpts(opts)
	if minLen == 0 {
		minLen = c.opts.MinSummaryLen
	}
	if maxLen == 0 {
		maxLen = c.opts.MaxSummaryLen
	}

	content = extractor.FilterNoneContentTags(content)
	blocks := extractor.GetBlocks(content)
	full := joinLines(blocks)
	expLen := summarizer.ResolveExpectedLength(blocks, opts.ExpectedLen, minLen, maxLen)
	sum := summarizer.New(c.seg, c.model, useShingle, filter)
	return sum.Summarize(full, expLen, title)
}

// Sentiment analyzes sentiment of content.
func (c *Client) Sentiment(content string, inject map[string]int) sentiment.Result {
	content = extractor.FilterNoneContentTags(content)
	blocks := extractor.GetBlocks(content)
	full := joinLines(blocks)
	return c.sentiment.Analyze(full, inject)
}

func (c *Client) resolveExtractOpts(opts ExtractOptions) (shingle bool, filter []string, minLen, maxLen int) {
	shingle = c.opts.Shingle
	if opts.Shingle != nil {
		shingle = *opts.Shingle
	}
	filter = c.opts.Filter
	if len(opts.Filter) > 0 {
		filter = opts.Filter
	}
	minLen = opts.MinSummaryLen
	maxLen = opts.MaxSummaryLen
	return
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}

// Package-level convenience functions using default client (must not Close default client).

func GetArticleSync(html string, expectedLen float64, shingle *bool) (*Article, error) {
	c, err := DefaultClient()
	if err != nil {
		return nil, err
	}
	return c.GetArticleSync(html, expectedLen, shingle)
}

func GetArticle(url string, expectedLen float64, shingle *bool) (*Article, error) {
	c, err := DefaultClient()
	if err != nil {
		return nil, err
	}
	return c.GetArticle(url, expectedLen, shingle)
}

func Summarize(content string, expectedLen float64, shingle *bool, title string) (string, error) {
	c, err := DefaultClient()
	if err != nil {
		return "", err
	}
	return c.Summarize(content, expectedLen, shingle, title), nil
}

func Sentiment(content string, inject map[string]int) (sentiment.Result, error) {
	c, err := DefaultClient()
	if err != nil {
		return sentiment.Result{}, err
	}
	return c.Sentiment(content, inject), nil
}
