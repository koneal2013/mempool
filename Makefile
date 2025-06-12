BIN_NAME := mempool
BIN_DIR := bin
CMD_DIR := cmd/mempool
PKG_DIR := pkg
MOCKS_DIR := mocks
MOCKGEN := $(shell go env GOPATH)/bin/mockgen

start:
	@$(MAKE) clean
	@$(MAKE) build
	@echo "running main program..."
	@./$(BIN_DIR)/$(BIN_NAME)

build:
	@$(MAKE) test
	@mkdir -p $(BIN_DIR)
	@cd $(CMD_DIR) && go build -o ../../$(BIN_DIR)/$(BIN_NAME) main.go

clean:
	@go clean -i ./...
	@rm -f $(BIN_DIR)/$(BIN_NAME)

# Run all tests in the project (including subpackages)
testq:
	@echo "running tests..."
	go test ./... -short

test:
	@$(MAKE) mocks
	@$(MAKE) testq

testqv:
	@echo "running tests..."
	go test -v ./...

testv:
	@$(MAKE) mocks
	@$(MAKE) testqv

cover:
	@mkdir -p .coverage || echo "hidden coverage folder exists"
	@go test -v -cover ./... -coverprofile .coverage/coverage.out
	@go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html

covero:
	@$(MAKE) cover
	@open .coverage/coverage.html

# Generate mocks for all non-test Go files in pkg/ and its subdirectories
files := $(filter-out %_test.go,$(shell find $(PKG_DIR) -type f -name '*.go'))

mocks:
	@rm -rf $(MOCKS_DIR)
	@go install github.com/golang/mock/mockgen@latest
	@mkdir -p $(MOCKS_DIR)
	@echo "generating mocks..."
	@$(foreach file, $(files), $(MOCKGEN) -package mocks -destination $(MOCKS_DIR)/$(subst /,_,${file}) -source ${file};)