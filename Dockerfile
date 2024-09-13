FROM alpine:latest
COPY /build/pve-status /main
RUN apk add --no-cache lm-sensors
ENTRYPOINT ["/main"]
CMD ["--config", "/config/config.json"]