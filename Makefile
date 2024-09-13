GO_BUILD=CGO_ENABLED=0 go build

.PHONY: init
init:
	go mod download

.PHONY: generate
generate:
	go generate ./...

.PHONY: run
run:
	go run -ldflags "-X main.BuildVersion=$(BUILD)" ./...

.PHONY: build
build:
	$(GO_BUILD) -o ./build/ ./...

.PHONY: buildLinuxX86
buildLinuxX86:
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o ./build/ ./...

.PHONY: buildImage
buildImage: buildLinuxX86
	docker buildx build --platform=linux/amd64 -t ghcr.io/tbxark-arc/pve-status:latest .
	docker push ghcr.io/tbxark-arc/pve-status:latest