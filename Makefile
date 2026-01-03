BINARY_NAME := www

.PHONY: build test clean

build:
	go build -o bin/$(BINARY_NAME) ./cmd/www

test:
	go test ./...

clean:
	rm -rf bin
