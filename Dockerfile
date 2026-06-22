# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o manga-web ./cmd/web

# Runtime stage
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /app/manga-web .
COPY web/ ./web/
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/ || exit 1
CMD ["./manga-web"]
