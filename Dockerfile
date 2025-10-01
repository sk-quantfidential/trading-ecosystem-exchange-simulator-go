FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy exchange-data-adapter-go dependency
COPY exchange-data-adapter-go/ ./exchange-data-adapter-go/

# Copy exchange-simulator-go files
COPY exchange-simulator-go/go.mod exchange-simulator-go/go.sum ./exchange-simulator-go/
WORKDIR /build/exchange-simulator-go
RUN go mod download

# Copy source and build
COPY exchange-simulator-go/ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o exchange-simulator ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates wget
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /build/exchange-simulator-go/exchange-simulator /app/exchange-simulator
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8082 9092

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8082/api/v1/health || exit 1

CMD ["./exchange-simulator"]
