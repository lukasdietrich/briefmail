VERSION  = $(shell git describe --always --dirty)

MAIN     = ./cmd/briefmail
INTERNAL = ./internal

TARGET   = ./target
SOURCE   = $(shell find . -name "*.go" ! -name "*_gen.go" ! -path "$(MOCKS)/*")

BINARY   = $(TARGET)/briefmail
WIRE     = $(MAIN)/wire_gen.go
MOCKS    = $(INTERNAL)/mocks
COVERAGE = $(TARGET)/coverage.txt

.PHONY: all
all: clean build test

.PHONY: clean
clean:
	rm -rf $(TARGET) $(WIRE) $(MOCKS)

.PHONY: build
build: $(BINARY)
	rice \
		--import-path github.com/lukasdietrich/briefmail/internal/database \
		append \
		--exec $(BINARY)

.PHONY: test
test: $(MOCKS)
	go test -race -coverprofile=$(COVERAGE) -covermode=atomic ./...

$(TARGET):
	mkdir -p $(TARGET)

$(BINARY): $(WIRE) | $(TARGET)
	go build -v -o $(BINARY) -ldflags '-X "main.Version=$(VERSION)"' $(MAIN)

$(WIRE): $(SOURCE)
	wire ./...

$(MOCKS): $(SOURCE)
	mockery \
		--dir $(INTERNAL) \
		--recursive \
		--name "^[A-Z]" \
		--output $(MOCKS) \
		--disable-version-string \
		--testonly \
		--keeptree
