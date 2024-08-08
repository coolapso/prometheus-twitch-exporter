build:
	 go build -o twitch-exporter
fmt:
	go fmt github.com/coolapso/prometheus-twitch-exporter/...
	
build-docker-multiarch:
	docker build --platform linux/arm/v7,linux/arm64/v8,linux/amd64 -t coolapso/twitch-exporter .
