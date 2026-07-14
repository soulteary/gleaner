package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/soulteary/gleaner"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gleaner", flag.ContinueOnError)
	fs.SetOutput(stderr)
	urlFlag := fs.String("url", "", "article URL to fetch")
	htmlFlag := fs.String("html", "", "path to HTML file")
	sizeFlag := fs.Int("size", 200, "expected summary length")
	shingleFlag := fs.Bool("shingle", false, "use shingle mode")
	jsonFlag := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	url := *urlFlag
	htmlPath := *htmlFlag
	size := *sizeFlag
	remaining := fs.Args()

	if url == "" && htmlPath == "" && len(remaining) >= 1 {
		if strings.HasPrefix(remaining[0], "http://") || strings.HasPrefix(remaining[0], "https://") {
			url = remaining[0]
		} else {
			htmlPath = remaining[0]
		}
	}
	if url != "" && len(remaining) >= 2 {
		if n, err := strconv.Atoi(remaining[1]); err == nil {
			size = n
		}
	}
	if htmlPath != "" && len(remaining) >= 2 {
		if n, err := strconv.Atoi(remaining[1]); err == nil {
			size = n
		}
	}

	if url == "" && htmlPath == "" {
		fmt.Fprintln(stderr, "usage: gleaner [-url URL|-html file] [-size N] [-shingle] [-json]")
		fmt.Fprintln(stderr, "       gleaner URL [size]")
		fmt.Fprintln(stderr, "       gleaner article.html [size]")
		return 2
	}

	client, err := gleaner.NewClient(gleaner.OptionsFromEnv())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer client.Close()

	opts := gleaner.DefaultExtractOptions(float64(size), shingleFlag)
	var article *gleaner.Article
	if htmlPath != "" {
		data, err := os.ReadFile(htmlPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		article, err = client.GetArticleSyncWithOptions(string(data), opts)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		article, err = client.GetArticle(url, float64(size), shingleFlag)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	if *jsonFlag {
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(article); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	fmt.Fprintf(stdout, "title: %s\n", article.Title)
	fmt.Fprintf(stdout, "pubdate: %s\n", article.Pubdate)
	fmt.Fprintf(stdout, "sentiment: %d\n", article.Sentiment)
	fmt.Fprintf(stdout, "summary: %s\n", article.Summary)
	fmt.Fprintf(stdout, "content:\n%s\n", article.Content)
	return 0
}

func init() {
	log.SetOutput(os.Stderr)
}
