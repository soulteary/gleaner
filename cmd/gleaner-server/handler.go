package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/soulteary/gleaner"
	"github.com/soulteary/gleaner/internal/fetcher"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB JSON payload

type requestBody struct {
	URL    string `json:"url"`
	HTML   string `json:"html"`
	Size   int    `json:"size"`
	Smooth *bool  `json:"smooth"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newHandler(client *gleaner.Client) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/", handleExtract(client))
	return mux
}

func handleExtract(client *gleaner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var req requestBody
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		if req.URL == "" && req.HTML == "" {
			writeError(w, http.StatusBadRequest, "url or html required")
			return
		}
		if req.URL != "" && req.HTML != "" {
			writeError(w, http.StatusBadRequest, "provide either url or html, not both")
			return
		}

		size := req.Size
		if size <= 0 {
			size = 200
		}
		opts := gleaner.DefaultExtractOptions(float64(size), req.Smooth)

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		var article *gleaner.Article
		if req.HTML != "" {
			article, err = client.GetArticleSyncWithOptions(req.HTML, opts)
		} else {
			article, err = client.GetArticleWithContext(ctx, req.URL, opts)
		}
		if err != nil {
			writeFetchError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(article)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

func writeFetchError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		writeError(w, http.StatusGatewayTimeout, msg)
	case errors.Is(err, fetcher.ErrBodyTooLarge):
		writeError(w, http.StatusBadGateway, msg)
	case errors.Is(err, fetcher.ErrBlockedURL):
		writeError(w, http.StatusBadRequest, msg)
	default:
		if strings.Contains(msg, "http status") {
			writeError(w, http.StatusBadGateway, msg)
			return
		}
		writeError(w, http.StatusBadGateway, msg)
	}
}
