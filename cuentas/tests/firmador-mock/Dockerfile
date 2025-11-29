FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy source
COPY main.go .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o firmador-mock main.go

# Final image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/firmador-mock .

EXPOSE 8113

CMD ["./firmador-mock"]
