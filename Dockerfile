# syntax=docker/dockerfile:1

FROM golang:1.27.3-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/gleaner-server ./cmd/gleaner-server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/gleaner-server /app/gleaner-server
EXPOSE 3824
ENV GLEANER_HOST=0.0.0.0
ENV GLEANER_PORT=3824
CMD ["/app/gleaner-server"]
