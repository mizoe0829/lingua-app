# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency definitions
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build pure Go binary without CGO
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server/main.go

# Run Stage: ultra-lightweight alpine container
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates for HTTPS/TLS requests (e.g. Gemini API)
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/server .

EXPOSE 8085

ENV PORT=8085
ENV DATABASE_PATH=/app/data/lingua.db

VOLUME /app/data

CMD ["./server"]

