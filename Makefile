BINARY_NAME=gotrack
CMD_DIR=./cmd/$(BINARY_NAME)
BIN_DIR=./bin

.PHONY: all run build install test clean

all: build

run:
	go run $(CMD_DIR)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

install:
	go install $(CMD_DIR)

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)