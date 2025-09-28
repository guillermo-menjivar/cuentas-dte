FROM golang:1.23-alpine AS builder

# Install git and build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cuentas main.go

# Final stage
FROM alpine:latest

# Create app directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/cuentas .
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Command to run
CMD ["./cuentas", "serve"]
