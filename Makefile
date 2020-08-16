VERSION  = $(shell git describe --always --dirty)

MAIN     = ./cmd/briefmail
INTERNAL = ./internal

TARGET   = ./target
SOURCE   = $(shell find . -name "*.go" ! -name "*_gen.go")

BINARY   = $(TARGET)/briefmail
WIRE     = $(MAIN)/wire_gen.go

.PHONY: all
all: clean build

.PHONY: clean
clean:
	rm -rf $(TARGET) $(WIRE)

.PHONY: build
build: $(BINARY)

$(TARGET):
	mkdir -p $(TARGET)

$(BINARY): $(WIRE) | $(TARGET)
	go build -o $(BINARY) -ldflags '-X "main.Version=$(VERSION)"' $(MAIN)

$(WIRE): $(SOURCE)
	wire ./...
