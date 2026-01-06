# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies for go-ethereum (CGO)
RUN apk add --no-cache gcc musl-dev linux-headers

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o faucet-backend .

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
