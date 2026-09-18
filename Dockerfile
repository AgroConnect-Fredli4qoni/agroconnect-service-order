FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service-order main.go

FROM alpine:3.18

RUN apk add --no-cache ca-certificates curl

WORKDIR /root/
COPY --from=builder /app/service-order .

EXPOSE 8082

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
  CMD curl -f http://localhost:8082/health || exit 1

CMD ["./service-order"]
