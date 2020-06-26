export GO111MODULE=on

SOURCE = $(wildcard */*.go) $(wildcard */*/*.go)
TARGET = ./target
BINARY = $(TARGET)/briefmail

.PHONY: all most clean tidy test lint build run

all: most lint

most: clean tidy build test

clean:
	-rm -r $(TARGET)

tidy:
	go mod tidy

test:
	go test -cover ./...

lint:
	go vet ./...
	golint ./...

build: $(BINARY)

run: $(BINARY)
	$(BINARY) \
		--verbose \
		start \
		--config _example/config.toml \
		--addressbook _example/addressbook.toml

$(BINARY): $(SOURCE)
	mkdir -p $(TARGET)
	wire ./...
	go build \
		-o $(BINARY) \
		-ldflags '-X "main.Version=$(shell git log -1 --pretty=format:"%h (%ai)")"' \
		./cmd/briefmail
