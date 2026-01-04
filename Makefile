.PHONY: build build-arm clean test install-deps

build:
	go build -o envedour-bot .

build-arm:
	GOOS=linux GOARCH=arm64 \
		go build \
		-ldflags="-s -w -extldflags '-Wl,--hash-style=gnu'" \
		-o envedour-bot-arm64 .

clean:
	rm -f envedour-bot envedour-bot-arm64

test:
	go test ./...

install-deps:
	go mod download
	go mod tidy
