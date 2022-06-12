SHELL=/bin/bash

PROJECT_NAME =$(notdir $(shell pwd))
LINTER := $(shell command -v ./build/golangci-lint 2> /dev/null)

COVERAGE_PROFILE ?= coverage.out

default: build

.PHONY: build
build:
	@echo "---> Building"
	go build -race -o ./build/$(PROJECT_NAME)

.PHONY: lint
lint:
	@echo "---> Linting"
ifndef LINTER
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build v1.46.0
endif
	@./build/golangci-lint run

.PHONY: run
run:
	@go run *.go

.PHONY: test
test:
	@echo "---> Testing"
	@ENVIRONMENT=test go test -p 1 ./... -coverprofile $(COVERAGE_PROFILE)

.PHONY: html
html: test
	@echo "---> Generating HTML coverage report"
	@go tool cover -html $(COVERAGE_PROFILE) -o ./build/coverage.html

.PHONY: clean
clean:
	@echo "---> Cleaning"
	rm -rf ./build
