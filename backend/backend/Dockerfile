FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/service-monitor .

FROM alpine:3.21
RUN adduser -D -H appuser
COPY --from=builder /out/service-monitor /usr/local/bin/service-monitor
USER appuser
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/service-monitor"]
