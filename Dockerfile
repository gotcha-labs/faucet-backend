# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary (no CGO)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o faucet-backend .

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/faucet-backend .

# Expose port
EXPOSE 3000

# Run
CMD ["./faucet-backend"]
