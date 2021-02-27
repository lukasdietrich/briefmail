VERSION  = $(shell git describe --always --dirty)

MAIN     = ./cmd/briefmail
INTERNAL = ./internal

TARGET   = ./target
SOURCE   = $(shell find . -name "*.go" ! -name "*_gen.go" ! -name "mock_*")
MOCKS    = $(shell find . -name "mock_*")

BINARY   = $(TARGET)/briefmail
WIRE     = $(MAIN)/wire_gen.go
COVERAGE = $(TARGET)/coverage.txt

.PHONY: all
all: clean build test

.PHONY: clean
clean:
	rm -rf $(TARGET) $(WIRE) $(MOCKS)

.PHONY: build
build: $(BINARY)

.PHONY: test
test: $(SOURCE)
	mockery \
		--dir $(INTERNAL) \
		--recursive \
		--name "^[A-Z]" \
		--disable-version-string \
		--inpackage
	go test -race -coverprofile=$(COVERAGE) -covermode=atomic ./...

$(TARGET):
	mkdir -p $(TARGET)

$(BINARY): $(WIRE) | $(TARGET)
	go build -v -o $(BINARY) -ldflags '-X "main.Version=$(VERSION)"' $(MAIN)

$(WIRE): $(SOURCE)
	wire ./...
