FROM golang:1.26-alpine AS builder

ENV PORT=8080
ENV GOPROXY="https://proxy.golang.org"
ENV CGO_ENABLED=0

WORKDIR /usr/src/app

# Cache dependency downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY *.go .
COPY static static
COPY templates templates

RUN go build -ldflags="-s -w" -o /usr/local/bin/server .

# ── Runtime image ─────────────────────────────────────────────────────────────
FROM alpine:3.23

LABEL org.opencontainers.image.source=https://github.com/icco/distraction.today
LABEL org.opencontainers.image.description="Some quotes"
LABEL org.opencontainers.image.licenses=MIT

RUN apk add --no-cache ca-certificates tzdata

EXPOSE 8080
ENV PORT=8080

# Run as a non-root user.
RUN adduser -S -u 1001 app
USER app

COPY --from=builder /usr/local/bin/server /usr/local/bin/server

CMD ["server"]
