VERSION  = $(shell git describe --always --dirty)

MAIN_PKG = ./cmd/briefmail
TARGET   = ./target
SOURCE   = $(shell find . -name "*.go" ! -name "*_gen.go")
BINARY   = $(TARGET)/briefmail
WIRE_GEN = $(MAIN_PKG)/wire_gen.go

.PHONY: all
all: clean build

.PHONY: clean
clean:
	rm -rf $(TARGET)
	rm -f $(WIRE_GEN)

.PHONY: build
build: $(BINARY)

$(WIRE_GEN): $(SOURCE)
	wire ./...

$(BINARY): $(SOURCE) $(WIRE_GEN)
	mkdir -p $(TARGET)
	go build \
		-o $(BINARY) \
		-ldflags '-X "main.Version=$(VERSION)"' \
		$(MAIN_PKG)
