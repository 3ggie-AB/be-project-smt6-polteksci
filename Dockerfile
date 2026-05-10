FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/netmonitor ./cmd/server

FROM alpine:3.22

RUN adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/netmonitor /app/netmonitor

USER appuser
EXPOSE 8080 5514/udp
ENTRYPOINT ["/app/netmonitor"]
