build-docker-multiarch:
	docker build --platform linux/arm/v7,linux/arm64/v8,linux/amd64 -t coolapso/twitch-exporter .
