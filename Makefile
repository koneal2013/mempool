start:
	@make clean
	@make build
	@echo "running main program..."
	@./mempool
build:
	@make test
	@go build

clean:
	@go clean -i

testq:
	@echo "running tests..."
	go test . ./pkg/**

test:
	@make mocks
	@make testq

testqv:
	@echo "running tests..."
	go test -v . ./pkg/**

testv:
	@make mocks
	@make testqv

cover:
	@mkdir .coverage || echo "hidden coverage folder exists"
	@go test -v -cover . ./pkg/** -coverprofile .coverage/coverage.out
	@go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html

covero:
	@make cover
	@open .coverage/coverage.html

files := $(filter-out %_test.go,$(wildcard pkg/**/*.go))

mocks:
	@rm -rf mocks
	@go get github.com/golang/mock/mockgen
	@echo "generating mocks..."
	@$(foreach file, $(files), mockgen -package mocks -destination mocks/$(subst /,_,$(file)) -source $(file);)