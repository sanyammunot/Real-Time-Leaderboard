# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files to backend directory
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download

# Copy source code (Context is root)
COPY . /app

# Build the binary
# We are in /app/backend, so build current directory
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main .

# Final Stage (Small image)
FROM alpine:latest

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run
CMD ["./main"]
