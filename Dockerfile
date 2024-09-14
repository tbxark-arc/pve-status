FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN make build

FROM alpine:latest
COPY --from=builder /app/build/pve-status /main
RUN apk add --no-cache lm-sensors
ENTRYPOINT ["/main"]
CMD ["--config", "/config.json"]