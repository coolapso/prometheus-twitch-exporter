FROM --platform=$BUILDPLATFORM golang:latest AS builder
ARG TARGETARCH
ARG VERSION

WORKDIR /twitch-exporter
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-X github.com/coolapso/prometheus-twitch-exporter/cmd.Version=${VERSION}" -a -o twitch-exporter

FROM alpine:latest

COPY --from=builder twitch-exporter/twitch-exporter /usr/bin/twitch-exporter

EXPOSE 9184
ENTRYPOINT ["/usr/bin/twitch-exporter"]
