# Stage 1: Builder
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go module definitions and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /app/server ./cmd/server

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/server /app/server

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/server"]
